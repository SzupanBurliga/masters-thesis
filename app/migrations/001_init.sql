CREATE TABLE users (
  id          BIGSERIAL PRIMARY KEY,
  email       TEXT NOT NULL UNIQUE,
  name        TEXT NOT NULL,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE categories (
  id         BIGSERIAL PRIMARY KEY,
  name       TEXT NOT NULL,
  parent_id  BIGINT REFERENCES categories(id)
);

CREATE TABLE products (
  id           BIGSERIAL PRIMARY KEY,
  name         TEXT NOT NULL,
  price_cents  INTEGER NOT NULL CHECK (price_cents >= 0),
  category_id  BIGINT NOT NULL REFERENCES categories(id)
);

CREATE TABLE orders (
  id          BIGSERIAL PRIMARY KEY,
  user_id     BIGINT NOT NULL REFERENCES users(id),
  status      TEXT NOT NULL CHECK (status IN ('NEW','PAID','SHIPPED','CANCELLED')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
  id          BIGSERIAL PRIMARY KEY,
  order_id    BIGINT NOT NULL REFERENCES orders(id),
  product_id  BIGINT NOT NULL REFERENCES products(id),
  quantity    INTEGER NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_orders_user    ON orders(user_id);
CREATE INDEX idx_orders_status  ON orders(status, created_at);
CREATE INDEX idx_items_order    ON order_items(order_id);
CREATE INDEX idx_products_cat   ON products(category_id);
