package env

import (
	"errors"
	"os"
	"strings"
)

const databaseURLEnvName = "DATABASE_URL"

type postgresConfig struct {
	uri string
}

func NewPostgresConfig() (*postgresConfig, error) {
	uri := strings.TrimSpace(os.Getenv(databaseURLEnvName))
	if uri == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	return &postgresConfig{uri: uri}, nil
}

func (cfg *postgresConfig) URI() string {
	return cfg.uri
}
