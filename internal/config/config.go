package config

import (
	"time"

	"github.com/Saveliy12/expenses/internal/config/env"
)

var _ Config = (*config)(nil)

type config struct {
	httpConf     HTTPConfig
	postgresConf PostgresConfig
	telegramConf TelegramConfig
	llmConf      LLMConfig
	timeZone     *time.Location
	seedDemoUser bool
}

func New() (*config, error) {
	httpConf, err := env.NewHTTPConfig()
	if err != nil {
		return nil, err
	}

	postgresConf, err := env.NewPostgresConfig()
	if err != nil {
		return nil, err
	}

	telegramConf, err := env.NewTelegramConfig()
	if err != nil {
		return nil, err
	}

	llmConf, err := env.NewLLMConfig()
	if err != nil {
		return nil, err
	}

	timeZone, err := env.NewTimeZone()
	if err != nil {
		return nil, err
	}

	seedDemoUser, err := env.NewSeedDemoUser()
	if err != nil {
		return nil, err
	}

	return &config{
		httpConf:     httpConf,
		postgresConf: postgresConf,
		telegramConf: telegramConf,
		llmConf:      llmConf,
		timeZone:     timeZone,
		seedDemoUser: seedDemoUser,
	}, nil
}

func (c *config) HTTP() HTTPConfig {
	return c.httpConf
}

func (c *config) Postgres() PostgresConfig {
	return c.postgresConf
}

func (c *config) Telegram() TelegramConfig {
	return c.telegramConf
}

func (c *config) LLM() LLMConfig {
	return c.llmConf
}

func (c *config) TimeZone() *time.Location {
	return c.timeZone
}

func (c *config) SeedDemoUser() bool {
	return c.seedDemoUser
}
