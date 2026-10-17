package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("not found")
	ErrInvalid  = errors.New("invalid argument")
)

const (
	DefaultListLimit   = 20
	DefaultOrdersLimit = 10
	MaxLimit           = 50
)

type User struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Category struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	ParentID *int64 `json:"parentId,omitempty"`
}

type Product struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PriceCents int32  `json:"price"`
	CategoryID int64  `json:"categoryId"`
}

type OrderItem struct {
	ID       int64    `json:"id"`
	OrderID  int64    `json:"-"`
	Quantity int32    `json:"quantity"`
	Product  *Product `json:"product,omitempty"`
}

type Order struct {
	ID        int64       `json:"id"`
	UserID    int64       `json:"userId"`
	Status    string      `json:"status"`
	CreatedAt time.Time   `json:"createdAt"`
	Items     []OrderItem `json:"items,omitempty"`
}

type UserWithOrders struct {
	User
	Orders []Order `json:"orders"`
}

type NewItem struct {
	ProductID int64 `json:"productId"`
	Quantity  int32 `json:"quantity"`
}

type NewOrder struct {
	UserID int64     `json:"userId"`
	Items  []NewItem `json:"items"`
}

// Service jest wspólną warstwą domenową dla REST, GraphQL i gRPC.
type Service interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	ListOrders(ctx context.Context, status string, limit, offset int) ([]Order, error)
	GetUserWithOrders(ctx context.Context, id int64, limit int) (*UserWithOrders, error)
	CreateOrder(ctx context.Context, in NewOrder) (*Order, error)

	// Metody hurtowe dla dataloaderów GraphQL.
	OrdersByUsers(ctx context.Context, userIDs []int64, limit int) (map[int64][]Order, error)
	ItemsByOrders(ctx context.Context, orderIDs []int64) (map[int64][]OrderItem, error)
	ProductsByIDs(ctx context.Context, ids []int64) (map[int64]*Product, error)
	CategoriesByIDs(ctx context.Context, ids []int64) (map[int64]*Category, error)
	UsersByIDs(ctx context.Context, ids []int64) (map[int64]*User, error)
}
