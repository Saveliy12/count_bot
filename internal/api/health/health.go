package health

import (
	"context"
	"net/http"
	"time"

	"github.com/Saveliy12/expenses/internal/api/httpx"
)

const readinessTimeout = 2 * time.Second

type Pinger interface {
	PingContext(ctx context.Context) error
}

type Handler struct {
	db Pinger
}

type response struct {
	Status string `json:"status"`
}

func NewHandler(db Pinger) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, response{Status: "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "not_ready", "database is unavailable")
		return
	}

	httpx.WriteJSON(w, http.StatusOK, response{Status: "ready"})
}
