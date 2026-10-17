package main

import (
	"context"
	"log"
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"

	"thesis/app/internal/config"
	gql "thesis/app/internal/graphql"
	"thesis/app/internal/graphql/generated"
	"thesis/app/internal/metrics"
	"thesis/app/internal/security"
	"thesis/app/internal/serve"
	"thesis/app/internal/store"
)

func main() {
	cfg := config.Load()
	cfg.Log("graphql")
	st, err := store.New(context.Background(), cfg.DBURL, cfg.PoolSize)
	if err != nil {
		log.Fatal(err)
	}
	metrics.Serve(cfg.MetricsAddr)

	// NewDefaultServer (jak w planie 9.4): cache sparsowanych zapytań, APQ, introspekcja.
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &gql.Resolver{Svc: st, Validate: cfg.Validation}}))
	if cfg.Complexity {
		srv.Use(gql.Limits{MaxDepth: cfg.MaxDepth, MaxCost: cfg.MaxCost, MaxAliases: cfg.MaxAliases})
	}

	var h http.Handler = gql.LoaderMiddleware(st, srv)
	// kolejność (od zewnątrz): metryki -> JWT -> rate limit -> rozmiar -> GraphQL
	if cfg.Validation {
		h = security.MaxBody(cfg.MaxBodyBytes)(h)
	}
	if cfg.RateLimit {
		h = security.RateLimit(security.NewLimiters(cfg.RateLimitRPS, cfg.RateLimitBurst))(h)
	}
	if cfg.Auth {
		h = security.Auth(security.NewJWT(cfg.JWTSecret))(h)
	}
	h = metrics.HTTP("graphql", func(*http.Request) string { return "graphql" })(h)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.Handle("/graphql", h)
	serve.HTTP(cfg, mux)
}
