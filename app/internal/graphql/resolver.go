package graphql

import "thesis/app/internal/domain"

// Validate włącza walidację argumentów (flaga VALIDATION_ENABLED, jak w REST i gRPC).
type Resolver struct {
	Svc      domain.Service
	Validate bool
}

func validLimit(l *int) bool { return l == nil || (*l >= 1 && *l <= domain.MaxLimit) }
