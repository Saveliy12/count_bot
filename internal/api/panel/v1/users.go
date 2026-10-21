package v1

import (
	"net/http"

	"github.com/Saveliy12/expenses/internal/api/httpx"
)

func (a *API) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.expenseService.ListFamilyUsers(r.Context())
	if err != nil {
		httpx.WriteServiceError(w, a.logger, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, usersResponse{Items: newUserResponses(users)})
}
