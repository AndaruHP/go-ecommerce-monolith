-- +goose Up
CREATE TABLE orders (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id),
    total_cents BIGINT NOT NULL CHECK (total_cents > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE order_items (
     id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
     order_id BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
     product_id BIGINT NOT NULL,
     product_name TEXT NOT NULL,
     price_cents BIGINT NOT NULL,
     quantity BIGINT NOT NULL CHECK (quantity > 0)
);

-- +goose Down
DROP TABLE order_items;
DROP TABLE orders;