package env

import (
	"os"
	"strings"
)

const (
	httpAddrEnvName = "HTTP_ADDR"
	httpAddrDefault = ":8083"
)

type httpConfig struct {
	addr string
}

func NewHTTPConfig() (*httpConfig, error) {
	addr := strings.TrimSpace(os.Getenv(httpAddrEnvName))
	if addr == "" {
		addr = httpAddrDefault
	}

	return &httpConfig{addr: addr}, nil
}

func (cfg *httpConfig) Addr() string {
	return cfg.addr
}
