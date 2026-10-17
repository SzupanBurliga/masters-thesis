package graphql

import (
	"context"
	"errors"
	"time"

	"github.com/vektah/gqlparser/v2/gqlerror"

	"thesis/app/internal/domain"
	"thesis/app/internal/graphql/generated"
	"thesis/app/internal/graphql/model"
)

// wrapErr: niezależny od protokołu model błędów (NOT_FOUND / INVALID w extensions).
func wrapErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &gqlerror.Error{Message: "not found", Extensions: map[string]any{"code": "NOT_FOUND"}}
	case errors.Is(err, domain.ErrInvalid):
		return &gqlerror.Error{Message: "invalid argument", Extensions: map[string]any{"code": "INVALID"}}
	}
	return err
}

func (r *categoryResolver) Parent(ctx context.Context, obj *domain.Category) (*domain.Category, error) {
	if obj.ParentID == nil {
		return nil, nil
	}
	return For(ctx).Categories.Load(ctx, *obj.ParentID)
}

func (r *mutationResolver) CreateOrder(ctx context.Context, input domain.NewOrder) (*domain.Order, error) {
	o, err := r.Svc.CreateOrder(ctx, input)
	return o, wrapErr(err)
}

func (r *orderResolver) Status(ctx context.Context, obj *domain.Order) (model.OrderStatus, error) {
	return model.OrderStatus(obj.Status), nil
}

func (r *orderResolver) CreatedAt(ctx context.Context, obj *domain.Order) (string, error) {
	return obj.CreatedAt.UTC().Format(time.RFC3339Nano), nil
}

func (r *orderResolver) User(ctx context.Context, obj *domain.Order) (*domain.User, error) {
	return For(ctx).Users.Load(ctx, obj.UserID)
}

func (r *orderResolver) Items(ctx context.Context, obj *domain.Order) ([]*domain.OrderItem, error) {
	items, err := For(ctx).Items.Load(ctx, obj.ID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.OrderItem, len(items))
	for i := range items {
		out[i] = &items[i]
	}
	return out, nil
}

func (r *productResolver) Category(ctx context.Context, obj *domain.Product) (*domain.Category, error) {
	return For(ctx).Categories.Load(ctx, obj.CategoryID)
}

func (r *queryResolver) User(ctx context.Context, id int64) (*domain.User, error) {
	if r.Validate && id <= 0 {
		return nil, wrapErr(domain.ErrInvalid)
	}
	u, err := r.Svc.GetUser(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, wrapErr(err)
	}
	return u, err
}

func (r *queryResolver) Orders(ctx context.Context, status *model.OrderStatus, limit *int, offset *int) ([]*domain.Order, error) {
	if r.Validate && (!validLimit(limit) || (offset != nil && *offset < 0)) {
		return nil, wrapErr(domain.ErrInvalid)
	}
	st := ""
	if status != nil {
		st = string(*status)
	}
	l, o := domain.DefaultListLimit, 0
	if limit != nil {
		l = *limit
	}
	if offset != nil {
		o = *offset
	}
	res, err := r.Svc.ListOrders(ctx, st, l, o)
	if err != nil {
		return nil, wrapErr(err)
	}
	out := make([]*domain.Order, len(res))
	for i := range res {
		out[i] = &res[i]
	}
	return out, nil
}

func (r *userResolver) Orders(ctx context.Context, obj *domain.User, limit *int) ([]*domain.Order, error) {
	if r.Validate && !validLimit(limit) {
		return nil, wrapErr(domain.ErrInvalid)
	}
	l := domain.DefaultOrdersLimit
	if limit != nil {
		l = *limit
	}
	if l > domain.MaxLimit {
		l = domain.MaxLimit
	}
	orders, err := For(ctx).Orders.Load(ctx, ordersKey{UserID: obj.ID, Limit: l})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Order, len(orders))
	for i := range orders {
		out[i] = &orders[i]
	}
	return out, nil
}

func (r *Resolver) Category() generated.CategoryResolver { return &categoryResolver{r} }
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }
func (r *Resolver) Order() generated.OrderResolver       { return &orderResolver{r} }
func (r *Resolver) Product() generated.ProductResolver   { return &productResolver{r} }
func (r *Resolver) Query() generated.QueryResolver       { return &queryResolver{r} }
func (r *Resolver) User() generated.UserResolver         { return &userResolver{r} }

type (
	categoryResolver struct{ *Resolver }
	mutationResolver struct{ *Resolver }
	orderResolver    struct{ *Resolver }
	productResolver  struct{ *Resolver }
	queryResolver    struct{ *Resolver }
	userResolver     struct{ *Resolver }
)
