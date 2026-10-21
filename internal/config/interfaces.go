package config

import "time"

type Config interface {
	HTTP() HTTPConfig
	Postgres() PostgresConfig
	Telegram() TelegramConfig
	LLM() LLMConfig
	TimeZone() *time.Location
}

type HTTPConfig interface {
	Addr() string
}

type PostgresConfig interface {
	URI() string
}

type TelegramConfig interface {
	Token() string
	Enabled() bool
}

type LLMConfig interface {
	APIKey() string
	Model() string
	BaseURL() string
	Timeout() time.Duration
	Enabled() bool
}
