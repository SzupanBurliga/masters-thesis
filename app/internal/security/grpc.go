package security

import (
	"context"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func AuthInterceptor(j *JWT) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		vals := md.Get("authorization")
		if len(vals) == 0 {
			return nil, status.Error(codes.Unauthenticated, "no token")
		}
		sub, err := j.Verify(strings.TrimPrefix(vals[0], "Bearer "))
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "bad token")
		}
		return h(WithSub(ctx, sub), req)
	}
}

func RateLimitInterceptor(l *Limiters) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		key := Sub(ctx)
		if key == "" {
			if p, ok := peer.FromContext(ctx); ok {
				key = p.Addr.String()
				if host, _, err := net.SplitHostPort(key); err == nil {
					key = host // sam adres IP, jak w REST (bez portu źródłowego)
				}
			}
		}
		if !l.Allow(key) {
			return nil, status.Error(codes.ResourceExhausted, "rate limit")
		}
		return h(ctx, req)
	}
}

// ValidationInterceptor wywołuje funkcję walidującą żądanie (zależną od typu wiadomości).
func ValidationInterceptor(validate func(req any) error) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
		if err := validate(req); err != nil {
			return nil, err
		}
		return h(ctx, req)
	}
}
