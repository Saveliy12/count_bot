package v1

import (
	"net/http"

	"github.com/Saveliy12/expenses/internal/api/httpx"
)

func (a *API) ListExpenses(w http.ResponseWriter, r *http.Request) {
	filter, err := parseExpenseFilter(r.URL.Query())
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	page, err := a.expenseService.ListFamilyExpenses(r.Context(), filter)
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, expensesResponse{
		Items:   newExpenseResponses(page.Items),
		HasMore: page.HasMore,
	})
}
