package product

import (
	"context"
	"errors"
	"go-monolith/internal/db"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalid   = errors.New("invalid product")
	ErrNotFound  = errors.New("product not found")
	ErrForbidden = errors.New("forbidden")
)

type Product struct {
	ID          int64     `json:"id"`
	SellerID    int64     `json:"seller_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Stock       int64     `json:"stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Input struct {
	Name        string
	Description string
	PriceCents  int64
	Stock       int64
}

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) Create(ctx context.Context, sellerID int64, in Input) (Product, error) {
	in, err := normalize(in)
	if err != nil {
		return Product{}, err
	}

	row, err := s.queries.CreateProduct(ctx, db.CreateProductParams{
		SellerID:    sellerID,
		Name:        in.Name,
		Description: in.Description,
		PriceCents:  in.PriceCents,
		Stock:       in.Stock,
	})
	if err != nil {
		return Product{}, err
	}
	return toProduct(row), nil
}

func (s *Service) List(ctx context.Context, limit, offset int32) ([]Product, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.queries.ListProducts(ctx, db.ListProductsParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]Product, 0, len(rows))
	for _, row := range rows {
		out = append(out, toProduct(row))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Product, error) {
	row, err := s.queries.GetProduct(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}
	return toProduct(row), err
}

func (s *Service) Update(ctx context.Context, userID, id int64, in Input) (Product, error) {
	in, err := normalize(in)
	if err != nil {
		return Product{}, err
	}
	curr, err := s.queries.GetProduct(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}
	if curr.SellerID != userID {
		return Product{}, ErrForbidden
	}
	row, err := s.queries.UpdateProduct(ctx, db.UpdateProductParams{
		ID:          id,
		Name:        in.Name,
		Description: in.Description,
		PriceCents:  in.PriceCents,
		Stock:       in.Stock,
		SellerID:    userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}
	return toProduct(row), nil
}

func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	curr, err := s.queries.GetProduct(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if curr.SellerID != userID {
		return ErrForbidden
	}
	n, err := s.queries.DeleteProduct(ctx, db.DeleteProductParams{
		ID:       id,
		SellerID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func normalize(in Input) (Input, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	if in.Name == "" || in.PriceCents <= 0 || in.Stock < 0 {
		return Input{}, ErrInvalid
	}

	return in, nil
}

func toProduct(row db.Product) Product {
	return Product{
		ID:          row.ID,
		SellerID:    row.SellerID,
		Name:        row.Name,
		Description: row.Description,
		PriceCents:  row.PriceCents,
		Stock:       row.Stock,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
