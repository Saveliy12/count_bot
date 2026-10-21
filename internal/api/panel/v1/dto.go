package v1

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

const dateLayout = "2006-01-02"

type expenseResponse struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Amount    string    `json:"amount"`
	Category  string    `json:"category"`
	SpentAt   string    `json:"spent_at"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type userResponse struct {
	TelegramID  int64  `json:"telegram_id"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}

type categoryResponse struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type categoriesResponse struct {
	Items []categoryResponse `json:"items"`
}

type expensesResponse struct {
	Items   []expenseResponse `json:"items"`
	HasMore bool              `json:"has_more"`
}

type usersResponse struct {
	Items []userResponse `json:"items"`
}

type createExpenseRequest struct {
	UserID   int64  `json:"user_id"`
	Amount   string `json:"amount"`
	Category string `json:"category"`
	SpentAt  string `json:"spent_at"`
	Comment  string `json:"comment"`
}

type updateExpenseRequest struct {
	Amount   *string `json:"amount"`
	Category *string `json:"category"`
	SpentAt  *string `json:"spent_at"`
	Comment  *string `json:"comment"`
}

func (r createExpenseRequest) input() (model.CreateExpenseInput, error) {
	amount, err := parseAmount(r.Amount)
	if err != nil {
		return model.CreateExpenseInput{}, err
	}

	category, err := model.ParseCategory(r.Category)
	if err != nil {
		return model.CreateExpenseInput{}, err
	}

	spentAt, err := parseDate(r.SpentAt, "spent_at")
	if err != nil {
		return model.CreateExpenseInput{}, err
	}

	return model.CreateExpenseInput{
		Amount:   amount,
		Category: category,
		SpentAt:  spentAt,
		Comment:  r.Comment,
	}, nil
}

func (r updateExpenseRequest) input() (model.UpdateExpenseInput, error) {
	var input model.UpdateExpenseInput

	if r.Amount != nil {
		amount, err := parseAmount(*r.Amount)
		if err != nil {
			return model.UpdateExpenseInput{}, err
		}
		input.Amount = &amount
	}
	if r.Category != nil {
		category, err := model.ParseCategory(*r.Category)
		if err != nil {
			return model.UpdateExpenseInput{}, err
		}
		input.Category = &category
	}
	if r.SpentAt != nil {
		spentAt, err := parseDate(*r.SpentAt, "spent_at")
		if err != nil {
			return model.UpdateExpenseInput{}, err
		}
		input.SpentAt = &spentAt
	}
	if r.Comment != nil {
		input.Comment = r.Comment
	}

	return input, nil
}

func parseExpenseFilter(values url.Values) (model.FamilyExpenseFilter, error) {
	var filter model.FamilyExpenseFilter

	if raw := strings.TrimSpace(values.Get("user_id")); raw != "" {
		userID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || userID <= 0 {
			return model.FamilyExpenseFilter{}, model.NewValidationError("user_id", "must be positive")
		}
		filter.UserID = userID
	}

	if raw := strings.TrimSpace(values.Get("from")); raw != "" {
		date, err := parseDate(raw, "from")
		if err != nil {
			return model.FamilyExpenseFilter{}, err
		}
		filter.From = date
	}
	if raw := strings.TrimSpace(values.Get("to")); raw != "" {
		date, err := parseDate(raw, "to")
		if err != nil {
			return model.FamilyExpenseFilter{}, err
		}
		filter.To = date
	}
	if raw := strings.TrimSpace(values.Get("category")); raw != "" {
		category, err := model.ParseCategory(raw)
		if err != nil {
			return model.FamilyExpenseFilter{}, err
		}
		filter.Category = &category
	}

	limit, err := parseNonNegativeInt(values.Get("limit"), "limit")
	if err != nil {
		return model.FamilyExpenseFilter{}, err
	}
	offset, err := parseNonNegativeInt(values.Get("offset"), "offset")
	if err != nil {
		return model.FamilyExpenseFilter{}, err
	}
	filter.Limit = limit
	filter.Offset = offset

	return filter, nil
}

func parseAmount(raw string) (decimal.Decimal, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return decimal.Zero, model.NewValidationError("amount", "is required")
	}

	amount, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, model.NewValidationError("amount", "must be a number")
	}

	return amount, nil
}

func parseDate(raw, field string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, model.NewValidationError(field, "is required")
	}

	date, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, model.NewValidationError(field, "must use YYYY-MM-DD format")
	}

	return date, nil
}

func parseNonNegativeInt(raw, field string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	number, err := strconv.Atoi(value)
	if err != nil || number < 0 {
		return 0, model.NewValidationError(field, "must be a non-negative integer")
	}

	return number, nil
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, model.NewValidationError("id", "must be positive")
	}

	return id, nil
}

func newExpenseResponse(expense model.Expense) expenseResponse {
	return expenseResponse{
		ID:        expense.ID,
		UserID:    expense.UserID,
		Amount:    expense.Amount.StringFixed(2),
		Category:  string(expense.Category),
		SpentAt:   expense.SpentAt.Format(dateLayout),
		Comment:   expense.Comment,
		CreatedAt: expense.CreatedAt,
		UpdatedAt: expense.UpdatedAt,
	}
}

func newExpenseResponses(expenses []model.Expense) []expenseResponse {
	result := make([]expenseResponse, 0, len(expenses))
	for _, expense := range expenses {
		result = append(result, newExpenseResponse(expense))
	}
	return result
}

func newUserResponses(users []model.User) []userResponse {
	result := make([]userResponse, 0, len(users))
	for _, user := range users {
		result = append(result, userResponse{
			TelegramID:  user.TelegramID,
			DisplayName: user.DisplayName,
			Username:    user.Username,
		})
	}
	return result
}
