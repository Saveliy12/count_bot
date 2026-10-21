package repository

import (
	"context"

	"github.com/Saveliy12/expenses/internal/model"
)

type UserRepository interface {
	Upsert(ctx context.Context, user model.User) (model.User, error)
	GetByTelegramID(ctx context.Context, telegramID int64) (model.User, error)
	ListFamilyMembers(ctx context.Context, familyID int64) ([]model.User, error)
}

type FamilyRepository interface {
	GetDefault(ctx context.Context) (model.Family, error)
}

type ExpenseRepository interface {
	Create(ctx context.Context, expense model.Expense) (model.Expense, error)
	GetByID(ctx context.Context, id int64) (model.Expense, error)
	ListByUser(ctx context.Context, userID int64, filter model.ExpenseFilter) (model.ExpensePage, error)
	ListByFamily(ctx context.Context, familyID int64, filter model.FamilyExpenseFilter) (model.ExpensePage, error)
	SummaryByUser(ctx context.Context, userID int64, filter model.ExpenseFilter) (model.ExpenseSummary, error)
	SummaryByFamily(ctx context.Context, familyID int64, filter model.FamilyExpenseFilter) (model.ExpenseSummary, error)
	LatestByUser(ctx context.Context, userID int64) (model.Expense, error)
	LatestByUserFiltered(ctx context.Context, userID int64, filter model.ExpenseFilter) (model.Expense, error)
	Update(ctx context.Context, expense model.Expense) (model.Expense, error)
	Delete(ctx context.Context, id int64) error
	DeleteIfUnchanged(ctx context.Context, expense model.Expense) error
}
