package v1

import (
	"net/http"

	"github.com/Saveliy12/expenses/internal/api/httpx"
)

func (a *API) DeleteExpense(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	if err := a.expenseService.DeleteFamilyExpense(r.Context(), id); err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
