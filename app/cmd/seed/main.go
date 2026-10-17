// Deterministyczny seeder (ziarno 42). Ładuje dane masowo przez CopyFrom.
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	nUsers      = 10000
	nCategories = 50
	nProducts   = 5000
	nOrders     = 50000
	nItems      = 200000
)

var statuses = []string{"NEW", "PAID", "SHIPPED", "CANCELLED"}

func main() {
	url := os.Getenv("DB_URL")
	if url == "" {
		url = "postgres://postgres:test@localhost:5432/shop"
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(ctx, `TRUNCATE order_items, orders, products, categories, users RESTART IDENTITY CASCADE`)
	must(err)

	r := rand.New(rand.NewSource(42))
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// users
	rows := make([][]any, nUsers)
	for i := range rows {
		id := i + 1
		rows[i] = []any{int64(id), fmt.Sprintf("user%d@example.com", id), fmt.Sprintf("User %d", id), base.Add(time.Duration(i) * time.Minute)}
	}
	copyRows(ctx, conn, "users", []string{"id", "email", "name", "created_at"}, rows)

	// categories: drzewo głębokości 4 (1 korzeń, 4 dzieci, reszta rozłożona na 3 kolejne poziomy)
	rows = make([][]any, nCategories)
	for i := range rows {
		id := int64(i + 1)
		var parent any
		switch {
		case i == 0:
		case i <= 4:
			parent = int64(1)
		default:
			parent = int64(2 + (i-5)%(i-1)) // zawsze wcześniejsza kategoria
		}
		rows[i] = []any{id, fmt.Sprintf("Category %d", id), parent}
	}
	copyRows(ctx, conn, "categories", []string{"id", "name", "parent_id"}, rows)

	// products
	rows = make([][]any, nProducts)
	for i := range rows {
		id := int64(i + 1)
		rows[i] = []any{id, fmt.Sprintf("Product %d", id), int32(100 + r.Intn(99900)), int64(1 + r.Intn(nCategories))}
	}
	copyRows(ctx, conn, "products", []string{"id", "name", "price_cents", "category_id"}, rows)

	// orders
	rows = make([][]any, nOrders)
	for i := range rows {
		id := int64(i + 1)
		rows[i] = []any{id, int64(1 + i%nUsers), statuses[r.Intn(len(statuses))], base.Add(time.Duration(i) * time.Second)}
	}
	copyRows(ctx, conn, "orders", []string{"id", "user_id", "status", "created_at"}, rows)

	// order_items: 4 na zamówienie
	rows = make([][]any, nItems)
	for i := range rows {
		rows[i] = []any{int64(i + 1), int64(1 + i%nOrders), int64(1 + r.Intn(nProducts)), int32(1 + r.Intn(5))}
	}
	copyRows(ctx, conn, "order_items", []string{"id", "order_id", "product_id", "quantity"}, rows)

	for _, t := range []string{"users", "categories", "products", "orders", "order_items"} {
		_, err = conn.Exec(ctx, fmt.Sprintf(`SELECT setval(pg_get_serial_sequence('%s','id'), (SELECT max(id) FROM %s))`, t, t))
		must(err)
	}
	_, err = conn.Exec(ctx, `ANALYZE`)
	must(err)

	for _, t := range []string{"users", "categories", "products", "orders", "order_items"} {
		var n int
		must(conn.QueryRow(ctx, `SELECT count(*) FROM `+t).Scan(&n))
		fmt.Printf("%-12s %d\n", t, n)
	}
	var sum string
	must(conn.QueryRow(ctx, `SELECT md5(string_agg(id::text || status || user_id::text, ',' ORDER BY id)) FROM orders`).Scan(&sum))
	fmt.Println("orders md5   ", sum)
}

func copyRows(ctx context.Context, c *pgx.Conn, table string, cols []string, rows [][]any) {
	_, err := c.CopyFrom(ctx, pgx.Identifier{table}, cols, pgx.CopyFromRows(rows))
	must(err)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
