-- +goose Up
CREATE TABLE products (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    seller_id BIGINT NOT NULL REFERENCES users (id),
    name TEXT NOT NULL ,
    description TEXT NOT NULL DEFAULT '',
    price_cents BIGINT NOT NULL CHECK ( price_cents > 0 ),
    stock BIGINT NOT NULL CHECK ( stock >= 0 ),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE products;