-- +goose Up
CREATE TABLE carts (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users (id)
);

CREATE TABLE cart_items (
    cart_id BIGINT NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    quantity BIGINT NOT NULL CHECK (quantity > 0),
    PRIMARY KEY (cart_id, product_id)
);

-- +goose Down
DROP TABLE cart_items;
DROP TABLE carts;