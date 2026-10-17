package main

import (
	"context"
	"log"

	"thesis/app/internal/config"
	"thesis/app/internal/metrics"
	"thesis/app/internal/rest"
	"thesis/app/internal/serve"
	"thesis/app/internal/store"
)

func main() {
	cfg := config.Load()
	cfg.Log("rest")
	st, err := store.New(context.Background(), cfg.DBURL, cfg.PoolSize)
	if err != nil {
		log.Fatal(err)
	}
	metrics.Serve(cfg.MetricsAddr)
	serve.HTTP(cfg, rest.New(st, cfg))
}
