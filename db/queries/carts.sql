-- name: GetCartByUser :one
SELECT id, user_id
FROM carts
WHERE user_id = $1;

-- name: EnsureCart :one
INSERT INTO carts (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO UPDATE
    SET user_id = EXCLUDED.user_id
RETURNING id, user_id;

-- name: ListCartItems :many
SELECT ci.product_id, p.name, p.price_cents, ci.quantity
FROM cart_items ci
         JOIN products p ON p.id = ci.product_id
WHERE ci.cart_id = $1
ORDER BY ci.product_id;

-- name: UpsertCartItem :exec
INSERT INTO cart_items (cart_id, product_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (cart_id, product_id) DO UPDATE
    SET quantity = EXCLUDED.quantity;

-- name: DeleteCartItem :exec
DELETE FROM cart_items
WHERE cart_id = $1 AND product_id = $2;