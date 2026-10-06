package auth

import (
	"context"
	"errors"
	"go-monolith/internal/db"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalid      = errors.New("invalid registration")
	ErrEmailTaken   = errors.New("email take")
	ErrUnauthorized = errors.New("unauthorized")
)

var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	queries *db.Queries
	secret  []byte
	ttl     time.Duration
}

func NewService(queries *db.Queries, secret []byte, ttl time.Duration) *Service {
	return &Service{queries: queries, secret: secret, ttl: ttl}
}

func (s *Service) Register(ctx context.Context, name, email, password string) (User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	if name == "" || !strings.Contains(email, "@") || len(password) < 8 {
		return User{}, ErrInvalid
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	row, err := s.queries.CreateUser(ctx, db.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}

	return User{
		ID:        row.ID,
		Name:      row.Name,
		Email:     row.Email,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	row, err := s.queries.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", ErrUnauthorized
	}

	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(password)); err != nil {
		return "", ErrUnauthorized
	}

	claims := jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(row.ID, 10),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.ttl)),
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", err
	}
	return signed, nil
}

func (s *Service) ParseToken(raw string) (int64, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return 0, ErrUnauthorized
	}

	id, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrUnauthorized
	}

	return id, nil
}
