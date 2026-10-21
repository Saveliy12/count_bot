package env

import (
	"os"
	"strings"
)

const telegramBotTokenEnvName = "TELEGRAM_BOT_TOKEN"

type telegramConfig struct {
	token string
}

func NewTelegramConfig() (*telegramConfig, error) {
	token := strings.TrimSpace(os.Getenv(telegramBotTokenEnvName))
	return &telegramConfig{token: token}, nil
}

func (cfg *telegramConfig) Token() string {
	return cfg.token
}

func (cfg *telegramConfig) Enabled() bool {
	return cfg.token != ""
}
