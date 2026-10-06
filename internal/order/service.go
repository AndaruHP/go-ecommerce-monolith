package order

import (
	"context"
	"errors"
	"go-monolith/internal/db"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmpty        = errors.New("cart empty")
	ErrInsufficient = errors.New("insufficient stock")
	ErrNotFound     = errors.New("not found")
)

type Item struct {
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	PriceCents  int64  `json:"price_cents"`
	Quantity    int64  `json:"quantity"`
	LineCents   int64  `json:"line_cents"`
}
type Order struct {
	ID         int64     `json:"id"`
	TotalCents int64     `json:"total_cents"`
	CreatedAt  time.Time `json:"created_at"`
	Items      []Item    `json:"items,omitempty"`
}
type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Checkout(ctx context.Context, userID int64) (Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	cart, err := q.GetCartByUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrEmpty
	}
	if err != nil {
		return Order{}, err
	}

	cartItems, err := q.ListCartItems(ctx, cart.ID)
	if err != nil {
		return Order{}, err
	}
	if len(cartItems) == 0 {
		return Order{}, ErrEmpty
	}

	lines := make([]Item, 0, len(cartItems))
	var total int64
	for _, item := range cartItems {
		product, err := q.GetProductForUpdate(ctx, item.ProductID)
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		if err != nil {
			return Order{}, err
		}
		if product.Stock < item.Quantity {
			return Order{}, ErrInsufficient
		}
		line := product.PriceCents * item.Quantity
		total += line
		lines = append(lines, Item{
			ProductID:   product.ID,
			ProductName: product.Name,
			PriceCents:  product.PriceCents,
			Quantity:    item.Quantity,
			LineCents:   line,
		})
	}

	row, err := q.CreateOrder(ctx, db.CreateOrderParams{
		UserID:     userID,
		TotalCents: total,
	})
	if err != nil {
		return Order{}, err
	}
	for _, line := range lines {
		if err := q.CreateOrderItem(ctx, db.CreateOrderItemParams{
			OrderID:     row.ID,
			ProductID:   line.ProductID,
			ProductName: line.ProductName,
			PriceCents:  line.PriceCents,
			Quantity:    line.Quantity,
		}); err != nil {
			return Order{}, err
		}
		n, err := q.DecreaseStock(ctx, db.DecreaseStockParams{
			Quantity: line.Quantity,
			ID:       line.ProductID,
		})
		if err != nil {
			return Order{}, err
		}
		if n == 0 {
			return Order{}, ErrInsufficient
		}
	}
	if err := q.ClearCart(ctx, cart.ID); err != nil {
		return Order{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Order{}, err
	}

	return Order{ID: row.ID, TotalCents: row.TotalCents, CreatedAt: row.CreatedAt.Time, Items: lines}, nil
}

func (s *Service) List(ctx context.Context, userID int64) ([]Order, error) {
	rows, err := db.New(s.pool).ListOrdersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Order, 0, len(rows))
	for _, row := range rows {
		out = append(out, Order{ID: row.ID, TotalCents: row.TotalCents, CreatedAt: row.CreatedAt.Time})
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, userID, orderID int64) (Order, error) {
	q := db.New(s.pool)
	row, err := q.GetOrder(ctx, db.GetOrderParams{ID: orderID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, err
	}
	itemRows, err := q.ListOrderItems(ctx, orderID)
	if err != nil {
		return Order{}, err
	}
	items := make([]Item, 0, len(itemRows))
	for _, item := range itemRows {
		items = append(items, Item{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			PriceCents:  item.PriceCents,
			Quantity:    item.Quantity,
			LineCents:   item.PriceCents * item.Quantity,
		})
	}
	return Order{ID: row.ID, TotalCents: row.TotalCents, CreatedAt: row.CreatedAt.Time, Items: items}, nil
}
