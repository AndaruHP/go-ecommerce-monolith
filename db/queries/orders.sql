-- name: GetProductForUpdate :one
SELECT id, name, price_cents, stock
FROM products
WHERE id = $1
    FOR UPDATE;

-- name: DecreaseStock :execrows
UPDATE products
SET stock = stock - sqlc.arg(quantity),
    updated_at = now()
WHERE id = sqlc.arg(id) AND stock >= sqlc.arg(quantity);

-- name: CreateOrder :one
INSERT INTO orders (user_id, total_cents)
VALUES ($1, $2)
RETURNING id, user_id, total_cents, created_at;

-- name: CreateOrderItem :exec
INSERT INTO order_items (order_id, product_id, product_name, price_cents, quantity)
VALUES ($1, $2, $3, $4, $5);

-- name: ClearCart :exec
DELETE FROM cart_items
WHERE cart_id = $1;

-- name: ListOrdersByUser :many
SELECT id, user_id, total_cents, created_at
FROM orders
WHERE user_id = $1
ORDER BY id DESC;

-- name: GetOrder :one
SELECT id, user_id, total_cents, created_at
FROM orders
WHERE id = $1 AND user_id = $2;

-- name: ListOrderItems :many
SELECT product_id, product_name, price_cents, quantity
FROM order_items
WHERE order_id = $1
ORDER BY id;