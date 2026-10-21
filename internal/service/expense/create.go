package expense

import (
	"context"
	"strings"

	"github.com/Saveliy12/expenses/internal/model"
)

func (s *Service) CreateExpense(
	ctx context.Context,
	actor model.Actor,
	input model.CreateExpenseInput,
) (model.Expense, error) {
	if err := validateActor(actor); err != nil {
		return model.Expense{}, err
	}

	user, err := s.users.GetByTelegramID(ctx, actor.TelegramID)
	if err != nil {
		return model.Expense{}, err
	}

	return s.createExpense(ctx, user, input)
}

func (s *Service) CreateFamilyExpense(
	ctx context.Context,
	userID int64,
	input model.CreateExpenseInput,
) (model.Expense, error) {
	if userID <= 0 {
		return model.Expense{}, model.NewValidationError("user_id", "must be positive")
	}

	user, err := s.familyUser(ctx, userID)
	if err != nil {
		return model.Expense{}, err
	}

	return s.createExpense(ctx, user, input)
}

func (s *Service) createExpense(
	ctx context.Context,
	user model.User,
	input model.CreateExpenseInput,
) (model.Expense, error) {
	now := s.now().UTC()
	expense := model.Expense{
		UserID:    user.TelegramID,
		Amount:    input.Amount,
		Category:  input.Category,
		SpentAt:   startOfDay(input.SpentAt),
		Comment:   strings.TrimSpace(input.Comment),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := model.ValidateExpense(expense); err != nil {
		return model.Expense{}, err
	}

	return s.expenses.Create(ctx, expense)
}
