package v1

import (
	"net/http"

	"github.com/Saveliy12/expenses/internal/api/httpx"
	"github.com/Saveliy12/expenses/internal/model"
)

func (a *API) ListCategories(w http.ResponseWriter, _ *http.Request) {
	categories := model.Categories()
	items := make([]categoryResponse, 0, len(categories))
	for _, category := range categories {
		items = append(items, categoryResponse{
			Value: string(category),
			Label: category.Label(),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, categoriesResponse{Items: items})
}
