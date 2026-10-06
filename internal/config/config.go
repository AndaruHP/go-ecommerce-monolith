package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr        string
	DatabaseURL string
	JWTSecret   string
	JWTTTL      time.Duration
}

// it has problem on reading .env, so rather only user os.GetEnv, i need to make it custom
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)
		if key != "" && os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func Load() (Config, error) {
	loadDotEnv()
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 characters")
	}

	ttl := 24 * time.Hour
	if raw := strings.TrimSpace(os.Getenv("JWT_TTL")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("JWT_TTL: %w", err)
		}
		ttl = parsed
	}

	return Config{
		Addr:        env("ADDR", ":8080"),
		DatabaseURL: env("DATABASE_URL", "postgres://app:app@localhost:5432/app?sslmode=disable"),
		JWTSecret:   secret,
		JWTTTL:      ttl,
	}, nil
}

func env(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}

	return v
}
