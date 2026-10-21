package expense

import (
	"context"

	"github.com/Saveliy12/expenses/internal/model"
)

func (s *Service) ListExpenses(
	ctx context.Context,
	actor model.Actor,
	filter model.ExpenseFilter,
) (model.ExpensePage, error) {
	if err := validateActor(actor); err != nil {
		return model.ExpensePage{}, err
	}

	filter, err := normalizeFilter(filter)
	if err != nil {
		return model.ExpensePage{}, err
	}

	return s.expenses.ListByUser(ctx, actor.TelegramID, filter)
}

func (s *Service) ListFamilyExpenses(
	ctx context.Context,
	filter model.FamilyExpenseFilter,
) (model.ExpensePage, error) {
	family, err := s.families.GetDefault(ctx)
	if err != nil {
		return model.ExpensePage{}, err
	}

	filter, err = normalizeFamilyFilter(filter)
	if err != nil {
		return model.ExpensePage{}, err
	}

	return s.expenses.ListByFamily(ctx, family.ID, filter)
}

func (s *Service) SummarizeExpenses(
	ctx context.Context,
	actor model.Actor,
	filter model.ExpenseFilter,
) (model.ExpenseSummary, error) {
	if err := validateActor(actor); err != nil {
		return model.ExpenseSummary{}, err
	}

	filter, err := normalizeFilter(filter)
	if err != nil {
		return model.ExpenseSummary{}, err
	}

	return s.expenses.SummaryByUser(ctx, actor.TelegramID, filter)
}

func (s *Service) LatestExpense(ctx context.Context, actor model.Actor) (model.Expense, error) {
	if err := validateActor(actor); err != nil {
		return model.Expense{}, err
	}

	return s.expenses.LatestByUser(ctx, actor.TelegramID)
}

func (s *Service) LatestExpenseFiltered(
	ctx context.Context,
	actor model.Actor,
	filter model.ExpenseFilter,
) (model.Expense, error) {
	if err := validateActor(actor); err != nil {
		return model.Expense{}, err
	}

	filter, err := normalizeFilter(filter)
	if err != nil {
		return model.Expense{}, err
	}

	return s.expenses.LatestByUserFiltered(ctx, actor.TelegramID, filter)
}
