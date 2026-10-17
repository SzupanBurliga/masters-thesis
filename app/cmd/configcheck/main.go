// configcheck sprawdza zachowanie wybranego interfejsu (macierz C0-C5, praktyczna.md 3.6).
// Wypisuje linie "check=true|false"; porównanie z oczekiwaniami robi infra/config_check.sh.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"thesis/app/internal/grpcsrv/pb"
)

var (
	iface  = flag.String("iface", "rest", "rest|graphql|grpc")
	addr   = flag.String("addr", "localhost:8443", "host:port")
	useTLS = flag.Bool("tls", false, "czy serwer ma TLS (klient użyje TLS)")
	token  = flag.String("token", "", "token JWT (puste = brak)")
)

// call zwraca kod w uniwersalnej postaci: "ok", "unauth", "ratelimit", "toolarge", "invalid", "other", "neterr".
type caller func(token string, payloadBytes int, deep bool) string

func main() {
	flag.Parse()
	var c caller
	var plain caller
	switch *iface {
	case "rest":
		c, plain = httpCaller(*useTLS, false), httpCaller(false, false)
	case "graphql":
		c, plain = httpCaller(*useTLS, true), httpCaller(false, true)
	case "grpc":
		c, plain = grpcCaller(*useTLS), grpcCaller(false)
	}

	// bazowe: poprawne wywołanie z tokenem
	fmt.Printf("baseline_ok=%v\n", c(*token, 0, false) == "ok")
	// TLS: połączenie bez TLS odrzucone (tylko sensowne, gdy serwer ma TLS)
	fmt.Printf("plain_rejected=%v\n", plain(*token, 0, false) != "ok")
	// brak tokenu
	fmt.Printf("no_token_401=%v\n", c("", 0, false) == "unauth")
	// duży ładunek 1 MB
	r := c(*token, 1<<20, false)
	fmt.Printf("big_payload_rejected=%v\n", r == "toolarge" || r == "invalid" || r == "neterr")
	// głębokie / złożone żądanie
	fmt.Printf("deep_rejected=%v\n", c(*token, 0, true) == "invalid")
	// rate limit: 600 równoległych wywołań
	var n429 int64
	var wg sync.WaitGroup
	sem := make(chan struct{}, 50)
	for i := 0; i < 600; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if c(*token, 0, false) == "ratelimit" {
				atomic.AddInt64(&n429, 1)
			}
		}()
	}
	wg.Wait()
	fmt.Printf("ratelimit_429=%v\n", n429 > 0)
}

func httpCaller(useTLS, graphql bool) caller {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	cl := &http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	return func(tok string, size int, deep bool) string {
		var req *http.Request
		switch {
		case graphql:
			q := `{ user(id:1){ id name } }`
			if deep {
				q = `{ user(id:1){ orders(limit:50){ items{ product{ category{ parent{ parent{ id } } } } } } } }`
			}
			if size > 0 {
				q = `{ user(id:1){ id } }` + strings.Repeat(" ", size)
			}
			b, _ := json.Marshal(map[string]any{"query": q})
			req, _ = http.NewRequest("POST", scheme+"://"+*addr+"/graphql", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
		case size > 0 || deep:
			var body string
			if deep {
				body = `{"userId":1,"items":[{"productId":1,"quantity":1}],"x":` + strings.Repeat(`{"a":`, 40) + "1" + strings.Repeat("}", 40) + "}"
			} else {
				body = `{"userId":1,"items":[{"productId":1,"quantity":1}],"pad":"` + strings.Repeat("a", size) + `"}`
			}
			req, _ = http.NewRequest("POST", scheme+"://"+*addr+"/orders", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
		default:
			req, _ = http.NewRequest("GET", scheme+"://"+*addr+"/users/1", nil)
		}
		if tok != "" {
			req.Header.Set("Authorization", "Bearer "+tok)
		}
		res, err := cl.Do(req)
		if err != nil {
			return "neterr"
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		switch res.StatusCode {
		case 200, 201:
			if graphql && bytes.Contains(body, []byte(`"errors"`)) {
				return "invalid"
			}
			return "ok"
		case 401:
			return "unauth"
		case 429:
			return "ratelimit"
		case 413:
			return "toolarge"
		case 400, 422:
			if graphql {
				return "invalid"
			}
			return "invalid"
		}
		return "other"
	}
}

func grpcCaller(useTLS bool) caller {
	var creds credentials.TransportCredentials = insecure.NewCredentials()
	if useTLS {
		creds = credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})
	}
	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		panic(err)
	}
	cl := pb.NewShopClient(conn)
	return func(tok string, size int, deep bool) string {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if tok != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+tok)
		}
		var err error
		switch {
		case size > 0:
			// duże pole tekstowe w żądaniu bez skutków ubocznych w bazie
			_, err = cl.ListOrders(ctx, &pb.ListOrdersRequest{Status: strings.Repeat("a", size), Limit: 1}, grpc.MaxCallSendMsgSize(16<<20))
		case deep:
			items := make([]*pb.NewItem, 1000)
			for i := range items {
				items[i] = &pb.NewItem{ProductId: 1, Quantity: 1}
			}
			_, err = cl.CreateOrder(ctx, &pb.CreateOrderRequest{UserId: 1, Items: items})
		default:
			_, err = cl.GetUser(ctx, &pb.GetUserRequest{Id: 1})
		}
		if err == nil {
			return "ok"
		}
		switch status.Code(err) {
		case codes.Unauthenticated:
			return "unauth"
		case codes.ResourceExhausted:
			if strings.Contains(err.Error(), "rate limit") {
				return "ratelimit"
			}
			return "toolarge"
		case codes.InvalidArgument:
			return "invalid"
		case codes.Unavailable:
			return "neterr"
		}
		return "other"
	}
}
