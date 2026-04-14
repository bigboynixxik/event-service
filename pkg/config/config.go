package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv    string `env:"APP_ENV" envDefault:"dev"`
	GRPCPort  string `env:"GRPC_PORT" envDefault:":50052"`
	JWTSecret string `env:"JWT_SECRET"`
	PGUser    string `env:"POSTGRES_USER" envDefault:"postgres"`
	PGPass    string `env:"POSTGRES_PASSWORD" envDefault:"postgres"`
	PGDb      string `env:"POSTGRES_DB" envDefault:"auth_db"`
	PGDSN     string `env:"PG_DSN"`
}

func LoadConfig(path string) (*Config, error) {
	err := godotenv.Load(path)
	if err != nil {
		return nil, fmt.Errorf("config.LoadConfig: %w", err)
	}
	var cfg Config

	err = env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("config.LoadConfig failed to parse config: %w", err)
	}
	return &cfg, nil
}
