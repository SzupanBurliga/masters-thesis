package grpcsrv

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"thesis/app/internal/domain"
	"thesis/app/internal/grpcsrv/pb"
)

var statuses = map[string]bool{"": true, "NEW": true, "PAID": true, "SHIPPED": true, "CANCELLED": true}

func invalid(msg string) error { return status.Error(codes.InvalidArgument, msg) }

// checkLimit: w proto3 limit=0 oznacza "brak" (wartość domyślna), więc w odróżnieniu od REST/GraphQL
// (limit < 1 jest błędem) gRPC go akceptuje. Znana, nieusuwalna różnica modeli.
func checkLimit(l int32) error {
	if l < 0 || l > domain.MaxLimit {
		return invalid("limit out of range")
	}
	return nil
}

// Validator zwraca funkcję walidującą: pola (validation) oraz długość repeated (complexity).
func Validator(validation, complexity bool, maxArray int) func(any) error {
	return func(req any) error {
		switch r := req.(type) {
		case *pb.GetUserRequest:
			if validation && r.Id <= 0 {
				return invalid("invalid id")
			}
		case *pb.GetUserWithOrdersRequest:
			if validation {
				if r.Id <= 0 {
					return invalid("invalid id")
				}
				return checkLimit(r.Limit)
			}
		case *pb.ListOrdersRequest:
			if validation {
				if !statuses[r.Status] || r.Offset < 0 {
					return invalid("invalid argument")
				}
				return checkLimit(r.Limit)
			}
		case *pb.CreateOrderRequest:
			if complexity && len(r.Items) > maxArray {
				return invalid("too many items")
			}
			if validation {
				if r.UserId <= 0 || len(r.Items) == 0 {
					return invalid("invalid order")
				}
				for _, it := range r.Items {
					if it.ProductId <= 0 || it.Quantity < 1 || it.Quantity > 1000 {
						return invalid("invalid item")
					}
				}
			}
		}
		return nil
	}
}
