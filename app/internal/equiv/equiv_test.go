package equiv

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"thesis/app/internal/config"
	gql "thesis/app/internal/graphql"
	"thesis/app/internal/graphql/generated"
	"thesis/app/internal/grpcsrv"
	"thesis/app/internal/grpcsrv/pb"
	"thesis/app/internal/rest"
	"thesis/app/internal/store"
)

// Wspólny, znormalizowany typ wyniku.
type item struct {
	Quantity int32
	Product  int64
	Name     string
	Price    int32
}
type order struct {
	ID     int64
	Status string
	Items  []item
}
type userOrders struct {
	ID, Name, Email string
	Orders          []order
}

type env struct {
	rest    *httptest.Server
	gql     *httptest.Server
	grpc    pb.ShopClient
	userIDs int
}

func setup(t *testing.T) *env {
	t.Helper()
	url := os.Getenv("DB_URL")
	if url == "" {
		url = "postgres://postgres:test@localhost:5432/shop"
	}
	st, err := store.New(context.Background(), url, 5)
	if err != nil {
		t.Skipf("brak bazy: %v", err)
	}
	t.Cleanup(st.Close)
	cfg := config.Config{}

	e := &env{userIDs: 10000}
	e.rest = httptest.NewServer(rest.New(st, cfg))
	t.Cleanup(e.rest.Close)

	srv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: &gql.Resolver{Svc: st}}))
	srv.AddTransport(transport.POST{})
	e.gql = httptest.NewServer(gql.LoaderMiddleware(st, srv))
	t.Cleanup(e.gql.Close)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := grpc.NewServer(grpcsrv.Options(cfg)...)
	pb.RegisterShopServer(gs, grpcsrv.New(st))
	go gs.Serve(lis)
	t.Cleanup(gs.Stop)
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	e.grpc = pb.NewShopClient(conn)
	return e
}

func get(t *testing.T, url string, v any) int {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		if err := json.NewDecoder(res.Body).Decode(v); err != nil {
			t.Fatal(err)
		}
	}
	return res.StatusCode
}

func gqlQuery(t *testing.T, base, q string, vars map[string]any, out any) []map[string]any {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"query": q, "variables": vars})
	res, err := http.Post(base, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var r struct {
		Data   json.RawMessage
		Errors []map[string]any
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		t.Fatal(err)
	}
	if len(r.Errors) == 0 {
		if err := json.Unmarshal(r.Data, out); err != nil {
			t.Fatal(err)
		}
	}
	return r.Errors
}

const o3Query = `query($id: ID!){ user(id:$id){ id name email orders(limit:10){ id status items{ quantity product{ id name price } } } } }`

func normOrders(os []order) []order {
	sort.Slice(os, func(i, j int) bool { return os[i].ID < os[j].ID })
	for i := range os {
		sort.Slice(os[i].Items, func(a, b int) bool {
			x, y := os[i].Items[a], os[i].Items[b]
			if x.Product != y.Product {
				return x.Product < y.Product
			}
			return x.Quantity < y.Quantity
		})
	}
	return os
}

func TestEquivO3(t *testing.T) {
	e := setup(t)
	r := rand.New(rand.NewSource(1))
	for n := 0; n < 100; n++ {
		id := 1 + r.Intn(e.userIDs)

		// REST
		var rr struct {
			ID     int64
			Name   string
			Email  string
			Orders []struct {
				ID     int64
				Status string
				Items  []struct {
					Quantity int32
					Product  struct {
						ID    int64
						Name  string
						Price int32
					}
				}
			}
		}
		if c := get(t, fmt.Sprintf("%s/users/%d/orders?expand=items.product&limit=10", e.rest.URL, id), &rr); c != 200 {
			t.Fatalf("rest %d", c)
		}
		a := userOrders{ID: fmt.Sprint(rr.ID), Name: rr.Name, Email: rr.Email}
		for _, o := range rr.Orders {
			oo := order{ID: o.ID, Status: o.Status}
			for _, it := range o.Items {
				oo.Items = append(oo.Items, item{it.Quantity, it.Product.ID, it.Product.Name, it.Product.Price})
			}
			a.Orders = append(a.Orders, oo)
		}

		// GraphQL
		var gr struct {
			User struct {
				ID     int64
				Name   string
				Email  string
				Orders []struct {
					ID     int64
					Status string
					Items  []struct {
						Quantity int32
						Product  struct {
							ID    int64
							Name  string
							Price int32
						}
					}
				}
			}
		}
		if errs := gqlQuery(t, e.gql.URL, o3Query, map[string]any{"id": id}, &gr); len(errs) > 0 {
			t.Fatalf("graphql %v", errs)
		}
		b := userOrders{ID: fmt.Sprint(gr.User.ID), Name: gr.User.Name, Email: gr.User.Email}
		for _, o := range gr.User.Orders {
			oo := order{ID: o.ID, Status: o.Status}
			for _, it := range o.Items {
				oo.Items = append(oo.Items, item{it.Quantity, it.Product.ID, it.Product.Name, it.Product.Price})
			}
			b.Orders = append(b.Orders, oo)
		}

		// gRPC
		pr, err := e.grpc.GetUserWithOrders(context.Background(), &pb.GetUserWithOrdersRequest{Id: int64(id), Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		c := userOrders{ID: fmt.Sprint(pr.Id), Name: pr.Name, Email: pr.Email}
		for _, o := range pr.Orders {
			oo := order{ID: o.Id, Status: o.Status}
			for _, it := range o.Items {
				oo.Items = append(oo.Items, item{it.Quantity, it.Product.Id, it.Product.Name, it.Product.PriceCents})
			}
			c.Orders = append(c.Orders, oo)
		}

		a.Orders, b.Orders, c.Orders = normOrders(a.Orders), normOrders(b.Orders), normOrders(c.Orders)
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		jc, _ := json.Marshal(c)
		if !bytes.Equal(ja, jb) || !bytes.Equal(jb, jc) {
			t.Fatalf("O3 id=%d niezgodne:\nrest=%s\ngql =%s\ngrpc=%s", id, ja, jb, jc)
		}
	}
}

func TestEquivO1O2(t *testing.T) {
	e := setup(t)
	r := rand.New(rand.NewSource(2))
	for n := 0; n < 100; n++ {
		id := 1 + r.Intn(e.userIDs)
		var ru struct {
			ID          int64
			Name, Email string
		}
		get(t, fmt.Sprintf("%s/users/%d", e.rest.URL, id), &ru)
		var gu struct {
			User struct {
				ID          int64
				Name, Email string
			}
		}
		gqlQuery(t, e.gql.URL, `query($id:ID!){ user(id:$id){ id name email } }`, map[string]any{"id": id}, &gu)
		pu, err := e.grpc.GetUser(context.Background(), &pb.GetUserRequest{Id: int64(id)})
		if err != nil {
			t.Fatal(err)
		}
		if ru.ID != gu.User.ID || ru.ID != pu.Id || ru.Name != gu.User.Name || ru.Name != pu.Name || ru.Email != gu.User.Email || ru.Email != pu.Email {
			t.Fatalf("O1 id=%d niezgodne: %+v %+v %+v", id, ru, gu, pu)
		}
	}
	for _, off := range []int{0, 20, 100} {
		var ro []struct {
			ID     int64
			Status string
		}
		get(t, fmt.Sprintf("%s/orders?status=PAID&limit=20&offset=%d", e.rest.URL, off), &ro)
		var go_ struct {
			Orders []struct {
				ID     int64
				Status string
			}
		}
		gqlQuery(t, e.gql.URL, fmt.Sprintf(`{ orders(status:PAID, limit:20, offset:%d){ id status } }`, off), nil, &go_)
		po, err := e.grpc.ListOrders(context.Background(), &pb.ListOrdersRequest{Status: "PAID", Limit: 20, Offset: int32(off)})
		if err != nil {
			t.Fatal(err)
		}
		if len(ro) != 20 || len(go_.Orders) != 20 || len(po.Orders) != 20 {
			t.Fatalf("O2 długości: %d %d %d", len(ro), len(go_.Orders), len(po.Orders))
		}
		for i := range ro {
			if ro[i].ID != go_.Orders[i].ID || ro[i].ID != po.Orders[i].Id || ro[i].Status != po.Orders[i].Status {
				t.Fatalf("O2 niezgodne na pozycji %d", i)
			}
		}
	}
}

func TestEquivNotFound(t *testing.T) {
	e := setup(t)
	var x any
	if c := get(t, e.rest.URL+"/users/99999999", &x); c != 404 {
		t.Fatalf("rest: %d", c)
	}
	var gu struct{ User *struct{ ID int64 } }
	errs := gqlQuery(t, e.gql.URL, `{ user(id:99999999){ id } }`, nil, &gu)
	if len(errs) == 0 {
		t.Fatal("graphql: oczekiwano błędu NOT_FOUND")
	}
	if code := errs[0]["extensions"].(map[string]any)["code"]; code != "NOT_FOUND" {
		t.Fatalf("graphql code=%v", code)
	}
	_, err := e.grpc.GetUser(context.Background(), &pb.GetUserRequest{Id: 99999999})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("grpc: %v", err)
	}
}
