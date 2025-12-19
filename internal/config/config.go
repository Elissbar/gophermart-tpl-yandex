package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	RunAddr     string `env:"RUN_ADDRESS"`
	DBURI       string `env:"DATABASE_URI"`
	AccrualAddr string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	JWTSecret   string `env:"JWTSecret"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("can't parse env: %w", err)
	}

	var runAddr, dbURI, accrualAddr, jwtSecret string
	flag.StringVar(&runAddr, "a", ":8080", ":<port>")
	flag.StringVar(&accrualAddr, "r", ":8081", ":<port>")
	flag.StringVar(&jwtSecret, "s", "jwt secret", "")
	flag.StringVar(&dbURI, "d", "", "Database connection string")
	// flag.StringVar(&dbURI, "d", "postgres://postgres:12345@localhost:5432/gophermart?sslmode=disable", "Database connection string")

	if cfg.RunAddr == "" {
		cfg.RunAddr = runAddr
	}
	if cfg.DBURI == "" {
		cfg.DBURI = dbURI
	}
	if cfg.AccrualAddr == "" {
		cfg.AccrualAddr = accrualAddr
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = jwtSecret
	}

	return &cfg, nil
}
