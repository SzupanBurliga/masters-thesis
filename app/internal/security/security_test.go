package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func ok() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = r.Body.Read(make([]byte, 1<<20))
		w.WriteHeader(200)
	})
}

func TestJWT(t *testing.T) {
	j := NewJWT("s")
	good, _ := j.Sign("u", time.Hour)
	expired, _ := j.Sign("u", -time.Hour)
	bad, _ := NewJWT("other").Sign("u", time.Hour)
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "u", "exp": time.Now().Add(time.Hour).Unix()}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	h := Auth(j)(ok())
	for name, c := range map[string]struct {
		tok  string
		want int
	}{"good": {good, 200}, "none": {"", 401}, "expired": {expired, 401}, "badsig": {bad, 401}, "alg-none": {none, 401}} {
		r := httptest.NewRequest("GET", "/", nil)
		if c.tok != "" {
			r.Header.Set("Authorization", "Bearer "+c.tok)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.want {
			t.Errorf("%s: %d != %d", name, w.Code, c.want)
		}
	}
}

func TestMaxBody(t *testing.T) {
	h := MaxBody(100)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 1000)
		n, err := r.Body.Read(buf)
		_ = n
		if err != nil && n == 0 {
			http.Error(w, "x", 413)
			return
		}
		w.WriteHeader(200)
	}))
	for size, want := range map[int]int{99: 200, 100: 200, 101: 413} {
		r := httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat("a", size)))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != want {
			t.Errorf("size %d: %d != %d", size, w.Code, want)
		}
	}
}

func TestRateLimit(t *testing.T) {
	h := RateLimit(NewLimiters(1, 5))(ok())
	got429 := 0
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
		if w.Code == 429 {
			got429++
		}
	}
	if got429 < 10 {
		t.Errorf("oczekiwano >=10 odpowiedzi 429, jest %d", got429)
	}
}

func TestCheckJSON(t *testing.T) {
	deep := strings.Repeat(`{"a":`, 20) + "1" + strings.Repeat("}", 20)
	if err := CheckJSON([]byte(deep), 8, 100); err == nil {
		t.Error("głębokość 20 powinna być odrzucona")
	}
	if err := CheckJSON([]byte(`{"a":{"b":[1,2,3]}}`), 8, 100); err != nil {
		t.Error(err)
	}
	arr := "[" + strings.Repeat("1,", 200) + "1]"
	if err := CheckJSON([]byte(arr), 8, 100); err == nil {
		t.Error("tablica 201 elementów powinna być odrzucona")
	}
}
