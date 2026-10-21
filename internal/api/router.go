package api

import (
	"log/slog"
	"net/http"

	"github.com/Saveliy12/expenses/internal/api/health"
	"github.com/Saveliy12/expenses/internal/api/httpx"
	panelv1 "github.com/Saveliy12/expenses/internal/api/panel/v1"
	"github.com/Saveliy12/expenses/internal/service"
)

func NewRouter(
	db health.Pinger,
	expenseService service.ExpenseService,
	logger *slog.Logger,
) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	healthHandler := health.NewHandler(db)
	panelAPI := panelv1.NewAPI(expenseService, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", httpx.AllowMethods(healthHandler.Health, http.MethodGet))
	mux.HandleFunc("/readyz", httpx.AllowMethods(healthHandler.Ready, http.MethodGet))

	mux.HandleFunc("/api/v1/users", httpx.AllowMethods(panelAPI.ListUsers, http.MethodGet))
	mux.HandleFunc("/api/v1/expenses", httpx.AllowMethods(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			panelAPI.ListExpenses(w, r)
		case http.MethodPost:
			panelAPI.CreateExpense(w, r)
		}
	}, http.MethodGet, http.MethodPost))
	mux.HandleFunc("/api/v1/expenses/{id}", httpx.AllowMethods(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			panelAPI.GetExpense(w, r)
		case http.MethodPatch:
			panelAPI.UpdateExpense(w, r)
		case http.MethodDelete:
			panelAPI.DeleteExpense(w, r)
		}
	}, http.MethodGet, http.MethodPatch, http.MethodDelete))

	mux.HandleFunc("/", notFound)

	return httpx.RequestID(
		httpx.RequestLogger(logger, httpx.Recoverer(logger, mux)),
	)
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteError(w, http.StatusNotFound, "not_found", "resource not found")
}
