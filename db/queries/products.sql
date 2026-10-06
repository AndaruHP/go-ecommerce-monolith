-- name: CreateProduct :one
INSERT INTO products (seller_id, name, description, price_cents, stock)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, seller_id, name, description, price_cents, stock, created_at, updated_at;

-- name: ListProducts :many
SELECT id, seller_id, name, description, price_cents, stock, created_at, updated_at
FROM products
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: GetProduct :one
SELECT id, seller_id, name, description, price_cents, stock, created_at, updated_at
FROM products
WHERE id = $1;

-- name: UpdateProduct :one
UPDATE products
SET name = $2,
    description = $3,
    price_cents = $4,
    stock = $5,
    updated_at = now()
WHERE id = $1 AND seller_id = $6
RETURNING id, seller_id, name, description, price_cents, stock, created_at, updated_at;

-- name: DeleteProduct :execrows
DELETE FROM products
WHERE id = $1 AND seller_id = $2;