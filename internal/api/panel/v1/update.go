package v1

import (
	"net/http"

	"github.com/Saveliy12/expenses/internal/api/httpx"
)

func (a *API) UpdateExpense(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	var request updateExpenseRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	input, err := request.input()
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	expense, err := a.expenseService.UpdateFamilyExpense(r.Context(), id, input)
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newExpenseResponse(expense))
}
