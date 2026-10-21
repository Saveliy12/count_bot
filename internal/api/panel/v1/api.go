package v1

import (
	"log/slog"

	"github.com/Saveliy12/expenses/internal/service"
)

type API struct {
	expenseService service.ExpenseService
	logger         *slog.Logger
}

func NewAPI(expenseService service.ExpenseService, logger *slog.Logger) *API {
	if logger == nil {
		logger = slog.Default()
	}

	return &API{
		expenseService: expenseService,
		logger:         logger,
	}
}
