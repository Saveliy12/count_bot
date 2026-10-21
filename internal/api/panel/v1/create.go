package v1

import (
	"fmt"
	"net/http"

	"github.com/Saveliy12/expenses/internal/api/httpx"
)

func (a *API) CreateExpense(w http.ResponseWriter, r *http.Request) {
	var request createExpenseRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	input, err := request.input()
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	expense, err := a.expenseService.CreateFamilyExpense(r.Context(), request.UserID, input)
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/expenses/%d", expense.ID))
	httpx.WriteJSON(w, http.StatusCreated, newExpenseResponse(expense))
}
