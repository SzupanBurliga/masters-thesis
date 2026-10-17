package grpcsrv

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"thesis/app/internal/config"
	"thesis/app/internal/domain"
	"thesis/app/internal/grpcsrv/pb"
	"thesis/app/internal/metrics"
	"thesis/app/internal/security"
)

type Server struct {
	pb.UnimplementedShopServer
	svc domain.Service
}

func New(svc domain.Service) *Server { return &Server{svc: svc} }

func toErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrNotFound):
		return status.Error(codes.NotFound, "not found")
	case errors.Is(err, domain.ErrInvalid):
		return status.Error(codes.InvalidArgument, "invalid argument")
	}
	return status.Error(codes.Internal, "internal error")
}

func conv(o domain.Order) *pb.Order {
	out := &pb.Order{Id: o.ID, UserId: o.UserID, Status: o.Status, CreatedAt: o.CreatedAt.UTC().Format(time.RFC3339Nano)}
	for _, it := range o.Items {
		pi := &pb.OrderItem{Id: it.ID, Quantity: it.Quantity}
		if it.Product != nil {
			pi.Product = &pb.Product{Id: it.Product.ID, Name: it.Product.Name, PriceCents: it.Product.PriceCents, CategoryId: it.Product.CategoryID}
		}
		out.Items = append(out.Items, pi)
	}
	return out
}

func (s *Server) GetUser(ctx context.Context, r *pb.GetUserRequest) (*pb.User, error) {
	u, err := s.svc.GetUser(ctx, r.Id)
	if err != nil {
		return nil, toErr(err)
	}
	return &pb.User{Id: u.ID, Name: u.Name, Email: u.Email}, nil
}

func (s *Server) ListOrders(ctx context.Context, r *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	os, err := s.svc.ListOrders(ctx, r.Status, int(r.Limit), int(r.Offset))
	if err != nil {
		return nil, toErr(err)
	}
	out := &pb.ListOrdersResponse{}
	for _, o := range os {
		out.Orders = append(out.Orders, conv(o))
	}
	return out, nil
}

func (s *Server) GetUserWithOrders(ctx context.Context, r *pb.GetUserWithOrdersRequest) (*pb.UserWithOrders, error) {
	u, err := s.svc.GetUserWithOrders(ctx, r.Id, int(r.Limit))
	if err != nil {
		return nil, toErr(err)
	}
	out := &pb.UserWithOrders{Id: u.ID, Name: u.Name, Email: u.Email}
	for _, o := range u.Orders {
		out.Orders = append(out.Orders, conv(o))
	}
	return out, nil
}

func (s *Server) CreateOrder(ctx context.Context, r *pb.CreateOrderRequest) (*pb.Order, error) {
	in := domain.NewOrder{UserID: r.UserId}
	for _, it := range r.Items {
		in.Items = append(in.Items, domain.NewItem{ProductID: it.ProductId, Quantity: it.Quantity})
	}
	o, err := s.svc.CreateOrder(ctx, in)
	if err != nil {
		return nil, toErr(err)
	}
	return conv(*o), nil
}

func metricsInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, h grpc.UnaryHandler) (any, error) {
	t := time.Now()
	resp, err := h(ctx, req)
	metrics.Observe("grpc", info.FullMethod, status.Code(err).String(), time.Since(t))
	return resp, err
}

// Options składa serwer gRPC. Kolejność interceptorów: metryki -> JWT -> rate limit -> walidacja.
func Options(cfg config.Config) []grpc.ServerOption {
	chain := []grpc.UnaryServerInterceptor{metricsInterceptor}
	if cfg.Auth {
		chain = append(chain, security.AuthInterceptor(security.NewJWT(cfg.JWTSecret)))
	}
	if cfg.RateLimit {
		chain = append(chain, security.RateLimitInterceptor(security.NewLimiters(cfg.RateLimitRPS, cfg.RateLimitBurst)))
	}
	if cfg.Validation || cfg.Complexity {
		chain = append(chain, security.ValidationInterceptor(Validator(cfg.Validation, cfg.Complexity, cfg.MaxArrayLen)))
	}
	opts := []grpc.ServerOption{grpc.ChainUnaryInterceptor(chain...)}
	if cfg.Validation {
		opts = append(opts, grpc.MaxRecvMsgSize(int(cfg.MaxBodyBytes)))
	} else {
		opts = append(opts, grpc.MaxRecvMsgSize(64<<20))
	}
	return opts
}
