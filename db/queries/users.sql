-- name: CreateUser :one
INSERT INTO users (email, password_hash, name)
VALUES ($1, $2, $3)
RETURNING id, email, password_hash, name, created_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, name, created_at
FROM users
WHERE email = $1;