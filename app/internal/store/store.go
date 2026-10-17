package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"thesis/app/internal/domain"
)

type Store struct{ pool *pgxpool.Pool }

func New(ctx context.Context, url string, poolSize int) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = int32(poolSize)
	cfg.MinConns = int32(poolSize)
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := p.Ping(ctx); err != nil {
		return nil, err
	}
	return &Store{pool: p}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Pool() *pgxpool.Pool { return s.pool }

var _ domain.Service = (*Store)(nil)

func clampLimit(limit, def int) int {
	if limit <= 0 {
		return def
	}
	if limit > domain.MaxLimit {
		return domain.MaxLimit
	}
	return limit
}

func (s *Store) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	u := &domain.User{}
	err := s.pool.QueryRow(ctx, `SELECT id, name, email FROM users WHERE id=$1`, id).Scan(&u.ID, &u.Name, &u.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return u, err
}

func (s *Store) ListOrders(ctx context.Context, status string, limit, offset int) ([]domain.Order, error) {
	limit = clampLimit(limit, domain.DefaultListLimit)
	if offset < 0 {
		return nil, domain.ErrInvalid
	}
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = s.pool.Query(ctx, `SELECT id, user_id, status, created_at FROM orders ORDER BY id LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		rows, err = s.pool.Query(ctx, `SELECT id, user_id, status, created_at FROM orders WHERE status=$1 ORDER BY id LIMIT $2 OFFSET $3`, status, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Order{}
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		o.CreatedAt = o.CreatedAt.UTC()
		out = append(out, o)
	}
	return out, rows.Err()
}

// GetUserWithOrders: stała liczba zapytań (4), niezależna od liczby zamówień.
func (s *Store) GetUserWithOrders(ctx context.Context, id int64, limit int) (*domain.UserWithOrders, error) {
	limit = clampLimit(limit, domain.DefaultOrdersLimit)
	u, err := s.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	res := &domain.UserWithOrders{User: *u, Orders: []domain.Order{}}
	om, err := s.OrdersByUsers(ctx, []int64{id}, limit)
	if err != nil {
		return nil, err
	}
	res.Orders = om[id]
	if res.Orders == nil {
		res.Orders = []domain.Order{}
		return res, nil
	}
	ids := make([]int64, len(res.Orders))
	for i, o := range res.Orders {
		ids[i] = o.ID
	}
	im, err := s.ItemsByOrders(ctx, ids)
	if err != nil {
		return nil, err
	}
	// produkty są dołączone w ItemsByOrders (JOIN), więc brak dodatkowego zapytania.
	for i := range res.Orders {
		res.Orders[i].Items = im[res.Orders[i].ID]
		if res.Orders[i].Items == nil {
			res.Orders[i].Items = []domain.OrderItem{}
		}
	}
	return res, nil
}

func (s *Store) OrdersByUsers(ctx context.Context, userIDs []int64, limit int) (map[int64][]domain.Order, error) {
	limit = clampLimit(limit, domain.DefaultOrdersLimit)
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, status, created_at FROM (
		  SELECT id, user_id, status, created_at,
		         row_number() OVER (PARTITION BY user_id ORDER BY id) AS rn
		  FROM orders WHERE user_id = ANY($1)
		) t WHERE rn <= $2 ORDER BY user_id, id`, userIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]domain.Order{}
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Status, &o.CreatedAt); err != nil {
			return nil, err
		}
		o.CreatedAt = o.CreatedAt.UTC()
		out[o.UserID] = append(out[o.UserID], o)
	}
	return out, rows.Err()
}

func (s *Store) ItemsByOrders(ctx context.Context, orderIDs []int64) (map[int64][]domain.OrderItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.order_id, i.quantity, p.id, p.name, p.price_cents, p.category_id
		FROM order_items i JOIN products p ON p.id = i.product_id
		WHERE i.order_id = ANY($1) ORDER BY i.order_id, i.id`, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64][]domain.OrderItem{}
	for rows.Next() {
		var it domain.OrderItem
		p := &domain.Product{}
		if err := rows.Scan(&it.ID, &it.OrderID, &it.Quantity, &p.ID, &p.Name, &p.PriceCents, &p.CategoryID); err != nil {
			return nil, err
		}
		it.Product = p
		out[it.OrderID] = append(out[it.OrderID], it)
	}
	return out, rows.Err()
}

func (s *Store) ProductsByIDs(ctx context.Context, ids []int64) (map[int64]*domain.Product, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, price_cents, category_id FROM products WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]*domain.Product{}
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.PriceCents, &p.CategoryID); err != nil {
			return nil, err
		}
		out[p.ID] = p
	}
	return out, rows.Err()
}

func (s *Store) CategoriesByIDs(ctx context.Context, ids []int64) (map[int64]*domain.Category, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, parent_id FROM categories WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]*domain.Category{}
	for rows.Next() {
		c := &domain.Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.ParentID); err != nil {
			return nil, err
		}
		out[c.ID] = c
	}
	return out, rows.Err()
}

func (s *Store) UsersByIDs(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, name, email FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]*domain.User{}
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
			return nil, err
		}
		out[u.ID] = u
	}
	return out, rows.Err()
}

func (s *Store) CreateOrder(ctx context.Context, in domain.NewOrder) (*domain.Order, error) {
	if in.UserID <= 0 || len(in.Items) == 0 {
		return nil, domain.ErrInvalid
	}
	for _, it := range in.Items {
		if it.Quantity <= 0 || it.ProductID <= 0 {
			return nil, domain.ErrInvalid
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	o := &domain.Order{UserID: in.UserID, Status: "NEW"}
	err = tx.QueryRow(ctx, `INSERT INTO orders(user_id, status) VALUES($1,'NEW') RETURNING id, created_at`, in.UserID).Scan(&o.ID, &o.CreatedAt)
	if err != nil {
		return nil, mapFK(err)
	}
	batch := &pgx.Batch{}
	for _, it := range in.Items {
		batch.Queue(`INSERT INTO order_items(order_id, product_id, quantity) VALUES($1,$2,$3) RETURNING id`, o.ID, it.ProductID, it.Quantity)
	}
	br := tx.SendBatch(ctx, batch)
	for _, it := range in.Items {
		var oi domain.OrderItem
		if err := br.QueryRow().Scan(&oi.ID); err != nil {
			br.Close()
			return nil, mapFK(err)
		}
		oi.Quantity = it.Quantity
		oi.Product = &domain.Product{ID: it.ProductID}
		o.Items = append(o.Items, oi)
	}
	if err := br.Close(); err != nil {
		return nil, mapFK(err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	o.CreatedAt = o.CreatedAt.UTC()
	return o, nil
}

// naruszenie klucza obcego (nieistniejący użytkownik/produkt) traktujemy jako błąd wejścia.
func mapFK(err error) error {
	var pe interface{ SQLState() string }
	if errors.As(err, &pe) && pe.SQLState() == "23503" {
		return domain.ErrInvalid
	}
	return err
}
