package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Saveliy12/expenses/internal/api"
	botrunner "github.com/Saveliy12/expenses/internal/bot"
	"github.com/Saveliy12/expenses/internal/config"
	"github.com/Saveliy12/expenses/internal/llm"
	"github.com/Saveliy12/expenses/internal/migrator"
	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository"
	repositoryexpense "github.com/Saveliy12/expenses/internal/repository/expense"
	repositoryfamily "github.com/Saveliy12/expenses/internal/repository/family"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
	repositoryuser "github.com/Saveliy12/expenses/internal/repository/user"
	"github.com/Saveliy12/expenses/internal/service"
	serviceexpense "github.com/Saveliy12/expenses/internal/service/expense"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	demoTelegramID    = 900000000001
)

type serviceProvider struct {
	config         config.Config
	db             *sql.DB
	users          repository.UserRepository
	families       repository.FamilyRepository
	expenses       repository.ExpenseRepository
	expenseService service.ExpenseService
	textProcessor  botrunner.TextProcessor
	httpServer     *http.Server
	botRunner      *botrunner.Runner
}

func newServiceProvider() *serviceProvider {
	return &serviceProvider{}
}

func (sp *serviceProvider) Config() (config.Config, error) {
	if sp.config == nil {
		cfg, err := config.New()
		if err != nil {
			return nil, err
		}
		sp.config = cfg
	}

	return sp.config, nil
}

func (sp *serviceProvider) DB(ctx context.Context) (*sql.DB, error) {
	if sp.db == nil {
		cfg, err := sp.Config()
		if err != nil {
			return nil, err
		}

		db, err := postgres.Open(ctx, cfg.Postgres().URI())
		if err != nil {
			return nil, err
		}

		if err := migrator.Up(ctx, db); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply migrations: %w", err)
		}

		sp.db = db
	}

	return sp.db, nil
}

func (sp *serviceProvider) UserRepository(ctx context.Context) (repository.UserRepository, error) {
	if sp.users == nil {
		db, err := sp.DB(ctx)
		if err != nil {
			return nil, err
		}
		sp.users = repositoryuser.NewRepository(db)
	}

	return sp.users, nil
}

func (sp *serviceProvider) FamilyRepository(ctx context.Context) (repository.FamilyRepository, error) {
	if sp.families == nil {
		db, err := sp.DB(ctx)
		if err != nil {
			return nil, err
		}
		sp.families = repositoryfamily.NewRepository(db)
	}

	return sp.families, nil
}

func (sp *serviceProvider) ExpenseRepository(ctx context.Context) (repository.ExpenseRepository, error) {
	if sp.expenses == nil {
		db, err := sp.DB(ctx)
		if err != nil {
			return nil, err
		}
		sp.expenses = repositoryexpense.NewRepository(db)
	}

	return sp.expenses, nil
}

func (sp *serviceProvider) ExpenseService(ctx context.Context) (service.ExpenseService, error) {
	if sp.expenseService == nil {
		users, err := sp.UserRepository(ctx)
		if err != nil {
			return nil, err
		}
		families, err := sp.FamilyRepository(ctx)
		if err != nil {
			return nil, err
		}
		expenses, err := sp.ExpenseRepository(ctx)
		if err != nil {
			return nil, err
		}

		sp.expenseService = serviceexpense.NewService(users, families, expenses)
	}

	return sp.expenseService, nil
}

func (sp *serviceProvider) TextProcessor(ctx context.Context) (botrunner.TextProcessor, error) {
	if sp.textProcessor == nil {
		cfg, err := sp.Config()
		if err != nil {
			return nil, err
		}
		if !cfg.LLM().Enabled() {
			slog.Warn("llm disabled", "reason", "GROQ_API_KEY is not set")
			sp.textProcessor = botrunner.StubProcessor{}
			return sp.textProcessor, nil
		}

		expenseService, err := sp.ExpenseService(ctx)
		if err != nil {
			return nil, err
		}

		client := llm.NewClient(llm.Config{
			BaseURL: cfg.LLM().BaseURL(),
			APIKey:  cfg.LLM().APIKey(),
			Model:   cfg.LLM().Model(),
			Timeout: cfg.LLM().Timeout(),
		}, slog.Default())
		sp.textProcessor = botrunner.NewLLMProcessor(
			client,
			expenseService,
			cfg.TimeZone(),
			slog.Default(),
		)
	}

	return sp.textProcessor, nil
}

func (sp *serviceProvider) SeedDemoUser(ctx context.Context) error {
	cfg, err := sp.Config()
	if err != nil {
		return err
	}
	if !cfg.SeedDemoUser() {
		return nil
	}

	expenseService, err := sp.ExpenseService(ctx)
	if err != nil {
		return err
	}

	users, err := expenseService.ListFamilyUsers(ctx)
	if err != nil {
		return err
	}
	if len(users) > 0 {
		return nil
	}

	if _, err := expenseService.EnsureUser(ctx, model.UserProfile{
		TelegramID:  demoTelegramID,
		DisplayName: "Демо-пользователь",
		Username:    "demo",
	}); err != nil {
		return fmt.Errorf("seed demo user: %w", err)
	}

	slog.Info("demo user seeded", "telegram_id", demoTelegramID)

	return nil
}

func (sp *serviceProvider) TelegramBot(ctx context.Context) (*botrunner.Runner, error) {
	if sp.botRunner == nil {
		cfg, err := sp.Config()
		if err != nil {
			return nil, err
		}
		if !cfg.Telegram().Enabled() {
			slog.Warn("telegram bot disabled", "reason", "TELEGRAM_BOT_TOKEN is not set")
			return nil, nil
		}

		expenseService, err := sp.ExpenseService(ctx)
		if err != nil {
			return nil, err
		}

		processor, err := sp.TextProcessor(ctx)
		if err != nil {
			return nil, err
		}

		handler := botrunner.NewHandler(expenseService, processor, slog.Default())
		runner, err := botrunner.NewRunner(ctx, cfg.Telegram().Token(), handler, slog.Default())
		if err != nil {
			return nil, err
		}

		sp.botRunner = runner
	}

	return sp.botRunner, nil
}

func (sp *serviceProvider) HTTPServer(ctx context.Context) (*http.Server, error) {
	if sp.httpServer == nil {
		cfg, err := sp.Config()
		if err != nil {
			return nil, err
		}
		db, err := sp.DB(ctx)
		if err != nil {
			return nil, err
		}
		expenseService, err := sp.ExpenseService(ctx)
		if err != nil {
			return nil, err
		}

		sp.httpServer = &http.Server{
			Addr:              cfg.HTTP().Addr(),
			Handler:           api.NewRouter(db, expenseService, slog.Default()),
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		}
	}

	return sp.httpServer, nil
}

func (sp *serviceProvider) Close() error {
	if sp.db != nil {
		return sp.db.Close()
	}
	return nil
}
