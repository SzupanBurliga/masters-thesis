package security

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

type subKey struct{}

// Sub zwraca tożsamość z kontekstu (ustawianą przez JWT middleware lub interceptor).
func Sub(ctx context.Context) string { s, _ := ctx.Value(subKey{}).(string); return s }

func WithSub(ctx context.Context, sub string) context.Context {
	return context.WithValue(ctx, subKey{}, sub)
}

// MaxBody: odrzuca (413) żądania większe niż n bajtów, a nieznany rozmiar ogranicza MaxBytesReader.
func MaxBody(n int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > n {
				http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, n)
			next.ServeHTTP(w, r)
		})
	}
}

// Auth: weryfikuje JWT; wyjątek dla wskazanych ścieżek (np. /healthz).
func Auth(j *JWT, skip ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range skip {
				if r.URL.Path == p {
					next.ServeHTTP(w, r)
					return
				}
			}
			tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
			sub, err := j.Verify(tok)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithSub(r.Context(), sub)))
		})
	}
}

// Limiters: token bucket per klucz. Mapa rośnie bez czyszczenia (ograniczenie opisane w pracy).
type Limiters struct {
	mu sync.Mutex
	m  map[string]*rate.Limiter
	r  rate.Limit
	b  int
}

func NewLimiters(rps float64, burst int) *Limiters {
	return &Limiters{m: map[string]*rate.Limiter{}, r: rate.Limit(rps), b: burst}
}

func (s *Limiters) Allow(key string) bool {
	s.mu.Lock()
	l, ok := s.m[key]
	if !ok {
		l = rate.NewLimiter(s.r, s.b)
		s.m[key] = l
	}
	s.mu.Unlock()
	return l.Allow()
}

func RateLimit(l *Limiters, skip ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range skip {
				if r.URL.Path == p {
					next.ServeHTTP(w, r)
					return
				}
			}
			key := Sub(r.Context())
			if key == "" {
				key, _, _ = net.SplitHostPort(r.RemoteAddr)
			}
			if !l.Allow(key) {
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
