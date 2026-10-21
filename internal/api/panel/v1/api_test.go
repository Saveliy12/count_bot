package v1

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/api/httpx"
	"github.com/Saveliy12/expenses/internal/model"
)

var apiTestNow = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Field   string `json:"field"`
	} `json:"error"`
}

type stubPanelService struct {
	users    []model.User
	usersErr error

	createdExpense model.Expense
	createErr      error
	createUserID   int64
	createInput    model.CreateExpenseInput

	expenses        []model.Expense
	expensesErr     error
	expensesHasMore bool
	listFilter      model.FamilyExpenseFilter
	summary         model.ExpenseSummary
	summaryErr      error

	expense model.Expense
	getErr  error

	updated     model.Expense
	updateErr   error
	updateID    int64
	updatePatch model.UpdateExpenseInput

	deleteErr error
	deleteID  int64
}

func (s *stubPanelService) ListFamilyUsers(context.Context) ([]model.User, error) {
	return s.users, s.usersErr
}

func (s *stubPanelService) CreateFamilyExpense(
	_ context.Context,
	userID int64,
	input model.CreateExpenseInput,
) (model.Expense, error) {
	s.createUserID = userID
	s.createInput = input
	return s.createdExpense, s.createErr
}

func (s *stubPanelService) ListFamilyExpenses(
	_ context.Context,
	filter model.FamilyExpenseFilter,
) (model.ExpensePage, error) {
	s.listFilter = filter
	return model.ExpensePage{Items: s.expenses, HasMore: s.expensesHasMore}, s.expensesErr
}

func (s *stubPanelService) SummarizeFamilyExpenses(
	_ context.Context,
	filter model.FamilyExpenseFilter,
) (model.ExpenseSummary, error) {
	s.listFilter = filter
	return s.summary, s.summaryErr
}

func (s *stubPanelService) GetFamilyExpense(context.Context, int64) (model.Expense, error) {
	return s.expense, s.getErr
}

func (s *stubPanelService) UpdateFamilyExpense(
	_ context.Context,
	id int64,
	patch model.UpdateExpenseInput,
) (model.Expense, error) {
	s.updateID = id
	s.updatePatch = patch
	return s.updated, s.updateErr
}

func (s *stubPanelService) DeleteFamilyExpense(_ context.Context, id int64) error {
	s.deleteID = id
	return s.deleteErr
}

func (s *stubPanelService) EnsureUser(context.Context, model.UserProfile) (model.User, error) {
	return model.User{}, errors.New("unexpected EnsureUser call")
}

func (s *stubPanelService) CreateExpense(
	context.Context,
	model.Actor,
	model.CreateExpenseInput,
) (model.Expense, error) {
	return model.Expense{}, errors.New("unexpected CreateExpense call")
}

func (s *stubPanelService) ListExpenses(
	context.Context,
	model.Actor,
	model.ExpenseFilter,
) (model.ExpensePage, error) {
	return model.ExpensePage{}, errors.New("unexpected ListExpenses call")
}

func (s *stubPanelService) SummarizeExpenses(
	context.Context,
	model.Actor,
	model.ExpenseFilter,
) (model.ExpenseSummary, error) {
	return model.ExpenseSummary{}, errors.New("unexpected SummarizeExpenses call")
}

func (s *stubPanelService) LatestExpense(context.Context, model.Actor) (model.Expense, error) {
	return model.Expense{}, errors.New("unexpected LatestExpense call")
}

func (s *stubPanelService) LatestExpenseFiltered(
	context.Context,
	model.Actor,
	model.ExpenseFilter,
) (model.Expense, error) {
	return model.Expense{}, errors.New("unexpected LatestExpenseFiltered call")
}

func (s *stubPanelService) GetExpense(
	context.Context,
	model.Actor,
	int64,
) (model.Expense, error) {
	return model.Expense{}, errors.New("unexpected GetExpense call")
}

func (s *stubPanelService) UpdateExpense(
	context.Context,
	model.Actor,
	int64,
	model.UpdateExpenseInput,
) (model.Expense, error) {
	return model.Expense{}, errors.New("unexpected UpdateExpense call")
}

func (s *stubPanelService) DeleteExpense(context.Context, model.Actor, int64) error {
	return errors.New("unexpected DeleteExpense call")
}

func (s *stubPanelService) DeleteExpenseIfUnchanged(
	context.Context,
	model.Actor,
	model.Expense,
) error {
	return errors.New("unexpected DeleteExpenseIfUnchanged call")
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newAPITestHandler(service *stubPanelService) http.Handler {
	api := NewAPI(service, discardLogger())

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/users", httpx.AllowMethods(api.ListUsers, http.MethodGet))
	mux.HandleFunc("/api/v1/categories", httpx.AllowMethods(api.ListCategories, http.MethodGet))
	mux.HandleFunc("/api/v1/expenses", httpx.AllowMethods(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.ListExpenses(w, r)
		case http.MethodPost:
			api.CreateExpense(w, r)
		}
	}, http.MethodGet, http.MethodPost))
	mux.HandleFunc("/api/v1/expenses/{id}", httpx.AllowMethods(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetExpense(w, r)
		case http.MethodPatch:
			api.UpdateExpense(w, r)
		case http.MethodDelete:
			api.DeleteExpense(w, r)
		}
	}, http.MethodGet, http.MethodPatch, http.MethodDelete))

	return mux
}

func TestListUsersAPI(t *testing.T) {
	service := &stubPanelService{
		users: []model.User{
			{TelegramID: 42, DisplayName: "Савелий", Username: "saveliy"},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var body usersResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].TelegramID != 42 {
		t.Fatalf("unexpected users response: %+v", body)
	}
}

func TestListCategoriesAPI(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(&stubPanelService{}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}

	var body categoriesResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 14 {
		t.Fatalf("expected 14 categories, got %d", len(body.Items))
	}

	foundFood := false
	for _, category := range body.Items {
		if category.Value == string(model.CategoryFood) {
			foundFood = true
			if category.Label != "еда" {
				t.Fatalf("unexpected food label %q", category.Label)
			}
		}
	}
	if !foundFood {
		t.Fatal("expected the food category")
	}
}

func TestListExpensesAPI(t *testing.T) {
	service := &stubPanelService{
		expenses:        []model.Expense{testExpense()},
		expensesHasMore: true,
		summary: model.ExpenseSummary{
			Total: decimal.RequireFromString("350.50"),
			Count: 1,
		},
	}
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/expenses?user_id=42&from=2026-10-01&to=2026-10-31&category=food&limit=10&offset=5",
		nil,
	)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if service.listFilter.UserID != 42 {
		t.Fatalf("expected user filter 42, got %d", service.listFilter.UserID)
	}
	if service.listFilter.Category == nil || *service.listFilter.Category != model.CategoryFood {
		t.Fatalf("unexpected category filter: %+v", service.listFilter.Category)
	}
	if service.listFilter.Limit != 10 || service.listFilter.Offset != 5 {
		t.Fatalf("unexpected pagination: %+v", service.listFilter)
	}
	if got := service.listFilter.From.Format(dateLayout); got != "2026-10-01" {
		t.Fatalf("unexpected from date %s", got)
	}

	var body expensesResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) != 1 || body.Items[0].Amount != "350.50" {
		t.Fatalf("unexpected expenses response: %+v", body)
	}
	if !body.HasMore {
		t.Fatal("expected has_more to be passed through")
	}
	if body.Total != "350.50" || body.Count != 1 {
		t.Fatalf("unexpected summary in response: total=%q count=%d", body.Total, body.Count)
	}
}

func TestCreateExpenseAPI(t *testing.T) {
	service := &stubPanelService{createdExpense: testExpense()}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/expenses",
		strings.NewReader(`{
			"user_id": 42,
			"amount": "350.5",
			"category": "food",
			"spent_at": "2026-10-19",
			"comment": " кофе "
		}`),
	)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	if location := response.Header().Get("Location"); location != "/api/v1/expenses/7" {
		t.Fatalf("unexpected Location header %q", location)
	}
	if service.createUserID != 42 {
		t.Fatalf("expected user ID 42, got %d", service.createUserID)
	}
	if !service.createInput.Amount.Equal(decimal.RequireFromString("350.5")) {
		t.Fatalf("unexpected amount: %s", service.createInput.Amount)
	}
	if service.createInput.Category != model.CategoryFood {
		t.Fatalf("unexpected category: %q", service.createInput.Category)
	}
	if body := response.Body.String(); !strings.Contains(body, `"amount":"350.50"`) {
		t.Fatalf("unexpected response body: %s", body)
	}
}

func TestListExpensesInvalidDate(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/expenses?from=not-a-date", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(&stubPanelService{}).ServeHTTP(response, request)

	assertValidationResponse(t, response, "from")
}

func TestCreateExpenseValidation(t *testing.T) {
	service := &stubPanelService{}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/expenses",
		strings.NewReader(`{
			"user_id": 42,
			"amount": "not-a-number",
			"category": "food",
			"spent_at": "2026-10-19"
		}`),
	)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	assertValidationResponse(t, response, "amount")
	if service.createUserID != 0 {
		t.Fatal("service must not be called for invalid input")
	}
}

func TestCreateExpenseRejectsUnknownFields(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/expenses",
		strings.NewReader(`{
			"user_id": 42,
			"amount": "350",
			"category": "food",
			"spent_at": "2026-10-19",
			"unknown": true
		}`),
	)
	response := httptest.NewRecorder()

	newAPITestHandler(&stubPanelService{}).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", response.Code)
	}
}

func TestGetExpenseAPI(t *testing.T) {
	service := &stubPanelService{expense: testExpense()}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/7", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
}

func TestGetExpenseInvalidID(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/abc", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(&stubPanelService{}).ServeHTTP(response, request)

	assertValidationResponse(t, response, "id")
}

func TestGetExpenseNotFound(t *testing.T) {
	service := &stubPanelService{getErr: model.ErrNotFound}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/7", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}

func TestUpdateExpenseAPI(t *testing.T) {
	service := &stubPanelService{updated: testExpense()}
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/expenses/7",
		strings.NewReader(`{"amount": "450", "comment": ""}`),
	)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if service.updateID != 7 {
		t.Fatalf("expected expense ID 7, got %d", service.updateID)
	}
	if service.updatePatch.Amount == nil || !service.updatePatch.Amount.Equal(decimal.NewFromInt(450)) {
		t.Fatalf("unexpected amount patch: %+v", service.updatePatch.Amount)
	}
	if service.updatePatch.Comment == nil || *service.updatePatch.Comment != "" {
		t.Fatalf("unexpected comment patch: %+v", service.updatePatch.Comment)
	}
}

func TestDeleteExpenseAPI(t *testing.T) {
	service := &stubPanelService{}
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/expenses/7", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
	if service.deleteID != 7 {
		t.Fatalf("expected deleted expense ID 7, got %d", service.deleteID)
	}
	if response.Body.Len() != 0 {
		t.Fatalf("expected an empty response body, got %q", response.Body.String())
	}
}

func TestAPIInternalError(t *testing.T) {
	service := &stubPanelService{getErr: errors.New("database failure")}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/expenses/7", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "database failure") {
		t.Fatal("internal error details must not be exposed")
	}
}

func TestUsersMethodNotAllowed(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	response := httptest.NewRecorder()

	newAPITestHandler(&stubPanelService{}).ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
}

func testExpense() model.Expense {
	return model.Expense{
		ID:        7,
		UserID:    42,
		Amount:    decimal.RequireFromString("350.50"),
		Category:  model.CategoryFood,
		SpentAt:   time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		Comment:   "кофе",
		CreatedAt: apiTestNow,
		UpdatedAt: apiTestNow,
	}
}

func assertValidationResponse(t *testing.T, response *httptest.ResponseRecorder, field string) {
	t.Helper()

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body.String())
	}

	var body errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "validation_error" {
		t.Fatalf("expected validation_error, got %q", body.Error.Code)
	}
	if body.Error.Field != field {
		t.Fatalf("expected field %q, got %q", field, body.Error.Field)
	}
}
