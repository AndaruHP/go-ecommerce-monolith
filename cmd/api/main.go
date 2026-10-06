package main

import (
	"context"
	"go-monolith/internal/auth"
	"go-monolith/internal/cart"
	"go-monolith/internal/config"
	"go-monolith/internal/db"
	"go-monolith/internal/order"
	"go-monolith/internal/product"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	queries := db.New(pool)
	authHandler := auth.NewHandler(auth.NewService(queries, []byte(cfg.JWTSecret), cfg.JWTTTL))
	productHandler := product.NewHandler(product.NewService(queries))
	cartHandler := cart.NewHandler(cart.NewService(queries))
	orderHandler := order.NewHandler(order.NewService(pool))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("POST /register", authHandler.Register)
	mux.HandleFunc("POST /login", authHandler.Login)
	mux.Handle("GET /me", authHandler.RequireUser(http.HandlerFunc(authHandler.Me)))

	mux.Handle("POST /products", authHandler.RequireUser(http.HandlerFunc(productHandler.Create)))
	mux.Handle("GET /products", authHandler.RequireUser(http.HandlerFunc(productHandler.List)))
	mux.Handle("GET /products/{id}", authHandler.RequireUser(http.HandlerFunc(productHandler.Get)))
	mux.Handle("PATCH /products/{id}", authHandler.RequireUser(http.HandlerFunc(productHandler.Update)))
	mux.Handle("DELETE /products/{id}", authHandler.RequireUser(http.HandlerFunc(productHandler.Delete)))

	mux.Handle("GET /cart", authHandler.RequireUser(http.HandlerFunc(cartHandler.Get)))
	mux.Handle("PUT /cart/items", authHandler.RequireUser(http.HandlerFunc(cartHandler.SetItem)))
	mux.Handle("DELETE /cart/items/{product_id}", authHandler.RequireUser(http.HandlerFunc(cartHandler.RemoveItem)))

	mux.Handle("POST /checkout", authHandler.RequireUser(http.HandlerFunc(orderHandler.Checkout)))
	mux.Handle("GET /orders", authHandler.RequireUser(http.HandlerFunc(orderHandler.List)))
	mux.Handle("GET /orders/{id}", authHandler.RequireUser(http.HandlerFunc(orderHandler.Get)))

	log.Printf("Listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatal(err)
	}
}
