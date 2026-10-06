package cart

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"go-monolith/internal/db"
)

var (
	ErrInvalid  = errors.New("invalid cart item")
	ErrNotFound = errors.New("product not found")
)

type Item struct {
	ProductID  int64  `json:"product_id"`
	Name       string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Quantity   int64  `json:"quantity"`
	LineCents  int64  `json:"line_cents"`
}

type Cart struct {
	Items      []Item `json:"items"`
	TotalCents int64  `json:"total_cents"`
}

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) Get(ctx context.Context, userID int64) (Cart, error) {
	row, err := s.queries.GetCartByUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return emptyCart(), nil
	}
	if err != nil {
		return Cart{}, err
	}
	return s.load(ctx, row.ID)
}

func (s *Service) SetItem(ctx context.Context, userID, productID, quantity int64) (Cart, error) {
	if productID <= 0 || quantity < 0 {
		return Cart{}, ErrInvalid
	}
	if quantity == 0 {
		return s.RemoveItem(ctx, userID, productID)
	}

	if _, err := s.queries.GetProduct(ctx, productID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Cart{}, ErrNotFound
		}
		return Cart{}, err
	}

	cart, err := s.queries.EnsureCart(ctx, userID)
	if err != nil {
		return Cart{}, err
	}
	if err := s.queries.UpsertCartItem(ctx, db.UpsertCartItemParams{
		CartID:    cart.ID,
		ProductID: productID,
		Quantity:  quantity,
	}); err != nil {
		return Cart{}, err
	}
	return s.load(ctx, cart.ID)
}

func (s *Service) RemoveItem(ctx context.Context, userID, productID int64) (Cart, error) {
	if productID <= 0 {
		return Cart{}, ErrInvalid
	}
	row, err := s.queries.GetCartByUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return emptyCart(), nil
	}
	if err != nil {
		return Cart{}, err
	}
	if err := s.queries.DeleteCartItem(ctx, db.DeleteCartItemParams{
		CartID:    row.ID,
		ProductID: productID,
	}); err != nil {
		return Cart{}, err
	}
	return s.load(ctx, row.ID)
}

func (s *Service) load(ctx context.Context, cartID int64) (Cart, error) {
	rows, err := s.queries.ListCartItems(ctx, cartID)
	if err != nil {
		return Cart{}, err
	}
	items := make([]Item, 0, len(rows))
	var total int64
	for _, row := range rows {
		line := row.PriceCents * row.Quantity
		total += line
		items = append(items, Item{
			ProductID:  row.ProductID,
			Name:       row.Name,
			PriceCents: row.PriceCents,
			Quantity:   row.Quantity,
			LineCents:  line,
		})
	}
	return Cart{Items: items, TotalCents: total}, nil
}

func emptyCart() Cart {
	return Cart{Items: []Item{}, TotalCents: 0}
}
