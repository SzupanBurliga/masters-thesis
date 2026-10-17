package main

import (
	"context"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"thesis/app/internal/config"
	"thesis/app/internal/grpcsrv"
	"thesis/app/internal/grpcsrv/pb"
	"thesis/app/internal/metrics"
	"thesis/app/internal/store"
)

func main() {
	cfg := config.Load()
	cfg.Log("grpc")
	st, err := store.New(context.Background(), cfg.DBURL, cfg.PoolSize)
	if err != nil {
		log.Fatal(err)
	}
	metrics.Serve(cfg.MetricsAddr)

	opts := grpcsrv.Options(cfg)
	if cfg.TLS {
		creds, err := credentials.NewServerTLSFromFile(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			log.Fatal(err)
		}
		opts = append(opts, grpc.Creds(creds))
	}
	s := grpc.NewServer(opts...)
	pb.RegisterShopServer(s, grpcsrv.New(st))
	hs := health.NewServer()
	healthpb.RegisterHealthServer(s, hs)
	reflection.Register(s)

	lis, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(s.Serve(lis))
}
