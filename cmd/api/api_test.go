package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-monolith/internal/config"
)

func TestMain(m *testing.M) {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if err := os.Chdir(dir); err != nil {
				panic(err)
			}
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found from " + dir)
		}
		dir = parent
	}
	os.Exit(m.Run())
}

func TestAPI(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(newMux(pool, []byte(cfg.JWTSecret), cfg.JWTTTL))
	t.Cleanup(srv.Close)
	c := &client{base: srv.URL, http: srv.Client()}

	stamp := time.Now().UnixNano()
	emailA := fmt.Sprintf("a%d@example.com", stamp)
	emailB := fmt.Sprintf("b%d@example.com", stamp)
	var tokenA, tokenB string
	var productID, scarceID, orderID int64

	t.Run("health", func(t *testing.T) {
		c.want(t, http.MethodGet, "/healthz", "", "", http.StatusOK)
	})

	t.Run("register rejects a short password", func(t *testing.T) {
		c.want(t, http.MethodPost, "/register", "", `{"name":"A","email":"short@example.com","password":"short"}`, http.StatusBadRequest)
	})
	t.Run("register rejects invalid json", func(t *testing.T) {
		c.want(t, http.MethodPost, "/register", "", `{`, http.StatusBadRequest)
	})
	t.Run("register", func(t *testing.T) {
		c.want(t, http.MethodPost, "/register", "", userJSON("Ada", emailA), http.StatusCreated)
	})
	t.Run("register rejects a duplicate email", func(t *testing.T) {
		c.want(t, http.MethodPost, "/register", "", userJSON("Ada", emailA), http.StatusConflict)
	})

	t.Run("login rejects a wrong password", func(t *testing.T) {
		c.want(t, http.MethodPost, "/login", "", `{"email":"`+emailA+`","password":"wrongpass"}`, http.StatusUnauthorized)
	})
	t.Run("login rejects an unknown email", func(t *testing.T) {
		c.want(t, http.MethodPost, "/login", "", `{"email":"missing@example.com","password":"secret123"}`, http.StatusUnauthorized)
	})
	t.Run("login", func(t *testing.T) {
		body := c.want(t, http.MethodPost, "/login", "", `{"email":"`+emailA+`","password":"secret123"}`, http.StatusOK)
		tokenA = jsonString(t, body, "access_token")
	})

	t.Run("me rejects a missing token", func(t *testing.T) {
		c.want(t, http.MethodGet, "/me", "", "", http.StatusUnauthorized)
	})
	t.Run("me rejects a bad token", func(t *testing.T) {
		c.want(t, http.MethodGet, "/me", "not-a-token", "", http.StatusUnauthorized)
	})
	t.Run("me", func(t *testing.T) {
		c.want(t, http.MethodGet, "/me", tokenA, "", http.StatusOK)
	})

	t.Run("create product rejects a missing token", func(t *testing.T) {
		c.want(t, http.MethodPost, "/products", "", productJSON("Mug", 5000, 5), http.StatusUnauthorized)
	})
	t.Run("create product rejects a non positive price", func(t *testing.T) {
		c.want(t, http.MethodPost, "/products", tokenA, productJSON("Mug", 0, 5), http.StatusBadRequest)
	})
	t.Run("create product", func(t *testing.T) {
		body := c.want(t, http.MethodPost, "/products", tokenA, productJSON("MacBook", 5000, 5), http.StatusCreated)
		productID = jsonInt(t, body, "id")
		if jsonInt(t, body, "stock") != 5 {
			t.Fatalf("stock: %s", body)
		}
	})
	t.Run("list products", func(t *testing.T) {
		body := c.want(t, http.MethodGet, "/products", tokenA, "", http.StatusOK)
		if !bytes.Contains(body, []byte(`"MacBook"`)) {
			t.Fatalf("list missing product: %s", body)
		}
	})
	t.Run("get product", func(t *testing.T) {
		c.want(t, http.MethodGet, "/products/"+itoa(productID), tokenA, "", http.StatusOK)
	})
	t.Run("get product rejects an unknown id", func(t *testing.T) {
		c.want(t, http.MethodGet, "/products/999999999", tokenA, "", http.StatusNotFound)
	})
	t.Run("get product rejects a bad id", func(t *testing.T) {
		c.want(t, http.MethodGet, "/products/nope", tokenA, "", http.StatusBadRequest)
	})

	t.Run("register second user", func(t *testing.T) {
		c.want(t, http.MethodPost, "/register", "", userJSON("Bob", emailB), http.StatusCreated)
		body := c.want(t, http.MethodPost, "/login", "", `{"email":"`+emailB+`","password":"secret123"}`, http.StatusOK)
		tokenB = jsonString(t, body, "access_token")
	})
	t.Run("update product rejects another user", func(t *testing.T) {
		c.want(t, http.MethodPatch, "/products/"+itoa(productID), tokenB, productJSON("Stolen", 1, 1), http.StatusForbidden)
	})
	t.Run("delete product rejects another user", func(t *testing.T) {
		c.want(t, http.MethodDelete, "/products/"+itoa(productID), tokenB, "", http.StatusForbidden)
	})
	t.Run("update product", func(t *testing.T) {
		body := c.want(t, http.MethodPatch, "/products/"+itoa(productID), tokenA, productJSON("MacBook", 5000, 5), http.StatusOK)
		if jsonString(t, body, "description") != "updated" {
			t.Fatalf("description: %s", body)
		}
	})
	t.Run("delete product", func(t *testing.T) {
		body := c.want(t, http.MethodPost, "/products", tokenA, productJSON("Cable", 1000, 1), http.StatusCreated)
		id := jsonInt(t, body, "id")
		c.want(t, http.MethodDelete, "/products/"+itoa(id), tokenA, "", http.StatusNoContent)
		c.want(t, http.MethodGet, "/products/"+itoa(id), tokenA, "", http.StatusNotFound)
	})
	t.Run("delete product rejects an unknown id", func(t *testing.T) {
		c.want(t, http.MethodDelete, "/products/999999999", tokenA, "", http.StatusNotFound)
	})

	t.Run("get cart is empty", func(t *testing.T) {
		body := c.want(t, http.MethodGet, "/cart", tokenA, "", http.StatusOK)
		if jsonInt(t, body, "total_cents") != 0 {
			t.Fatalf("cart: %s", body)
		}
	})
	t.Run("add cart item rejects an unknown product", func(t *testing.T) {
		c.want(t, http.MethodPut, "/cart/items", tokenA, `{"product_id":999999999,"quantity":1}`, http.StatusNotFound)
	})
	t.Run("add cart item rejects a negative quantity", func(t *testing.T) {
		c.want(t, http.MethodPut, "/cart/items", tokenA, fmt.Sprintf(`{"product_id":%d,"quantity":-1}`, productID), http.StatusBadRequest)
	})
	t.Run("add cart item", func(t *testing.T) {
		body := c.want(t, http.MethodPut, "/cart/items", tokenA, fmt.Sprintf(`{"product_id":%d,"quantity":2}`, productID), http.StatusOK)
		if jsonInt(t, body, "total_cents") != 10000 {
			t.Fatalf("cart: %s", body)
		}
	})
	t.Run("remove cart item", func(t *testing.T) {
		body := c.want(t, http.MethodDelete, "/cart/items/"+itoa(productID), tokenA, "", http.StatusOK)
		if jsonInt(t, body, "total_cents") != 0 {
			t.Fatalf("cart: %s", body)
		}
	})
	t.Run("remove cart item rejects a bad id", func(t *testing.T) {
		c.want(t, http.MethodDelete, "/cart/items/nope", tokenA, "", http.StatusBadRequest)
	})

	t.Run("checkout rejects an empty cart", func(t *testing.T) {
		c.want(t, http.MethodPost, "/checkout", tokenA, "", http.StatusBadRequest)
	})
	t.Run("checkout", func(t *testing.T) {
		c.want(t, http.MethodPut, "/cart/items", tokenA, fmt.Sprintf(`{"product_id":%d,"quantity":2}`, productID), http.StatusOK)
		body := c.want(t, http.MethodPost, "/checkout", tokenA, "", http.StatusCreated)
		orderID = jsonInt(t, body, "id")
		if jsonInt(t, body, "total_cents") != 10000 {
			t.Fatalf("order: %s", body)
		}
	})
	t.Run("cart is empty after checkout", func(t *testing.T) {
		body := c.want(t, http.MethodGet, "/cart", tokenA, "", http.StatusOK)
		if jsonInt(t, body, "total_cents") != 0 {
			t.Fatalf("cart: %s", body)
		}
	})
	t.Run("checkout decrements stock", func(t *testing.T) {
		body := c.want(t, http.MethodGet, "/products/"+itoa(productID), tokenA, "", http.StatusOK)
		if jsonInt(t, body, "stock") != 3 {
			t.Fatalf("stock: %s", body)
		}
	})
	t.Run("list orders", func(t *testing.T) {
		body := c.want(t, http.MethodGet, "/orders", tokenA, "", http.StatusOK)
		if !bytes.Contains(body, []byte(itoa(orderID))) {
			t.Fatalf("orders: %s", body)
		}
	})
	t.Run("get order", func(t *testing.T) {
		body := c.want(t, http.MethodGet, "/orders/"+itoa(orderID), tokenA, "", http.StatusOK)
		if jsonInt(t, body, "total_cents") != 10000 {
			t.Fatalf("order: %s", body)
		}
	})
	t.Run("get order hides another user's order", func(t *testing.T) {
		c.want(t, http.MethodGet, "/orders/"+itoa(orderID), tokenB, "", http.StatusNotFound)
	})
	t.Run("get order rejects a bad id", func(t *testing.T) {
		c.want(t, http.MethodGet, "/orders/nope", tokenA, "", http.StatusBadRequest)
	})

	t.Run("checkout rejects insufficient stock", func(t *testing.T) {
		body := c.want(t, http.MethodPost, "/products", tokenA, productJSON("Last", 100, 1), http.StatusCreated)
		scarceID = jsonInt(t, body, "id")
		c.want(t, http.MethodPut, "/cart/items", tokenA, fmt.Sprintf(`{"product_id":%d,"quantity":2}`, scarceID), http.StatusOK)
		c.want(t, http.MethodPost, "/checkout", tokenA, "", http.StatusConflict)
		body = c.want(t, http.MethodGet, "/products/"+itoa(scarceID), tokenA, "", http.StatusOK)
		if jsonInt(t, body, "stock") != 1 {
			t.Fatalf("stock changed: %s", body)
		}
	})
}

func userJSON(name, email string) string {
	return fmt.Sprintf(`{"name":%q,"email":%q,"password":"secret123"}`, name, email)
}

func productJSON(name string, price, stock int64) string {
	description := "bagus ini"
	if name == "MacBook" && price == 5000 && stock == 5 {
		description = "updated"
	}
	return fmt.Sprintf(`{"name":%q,"description":%q,"price_cents":%d,"stock":%d}`, name, description, price, stock)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

type client struct {
	base string
	http *http.Client
}

func (c *client) want(t *testing.T, method, path, token, body string, status int) []byte {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, c.base+path, rdr)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	got, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != status {
		t.Fatalf("%s %s: got %d %s, want %d", method, path, res.StatusCode, bytes.TrimSpace(got), status)
	}
	return got
}

func jsonString(t *testing.T, body []byte, key string) string {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	s, _ := m[key].(string)
	if s == "" {
		t.Fatalf("missing %s in %s", key, body)
	}
	return s
}

func jsonInt(t *testing.T, body []byte, key string) int64 {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	n, ok := m[key].(float64)
	if !ok {
		t.Fatalf("missing %s in %s", key, body)
	}
	return int64(n)
}
