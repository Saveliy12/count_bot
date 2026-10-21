package service

import (
	"context"

	"github.com/Saveliy12/expenses/internal/model"
)

type ExpenseService interface {
	EnsureUser(ctx context.Context, profile model.UserProfile) (model.User, error)
	ListFamilyUsers(ctx context.Context) ([]model.User, error)

	CreateExpense(ctx context.Context, actor model.Actor, input model.CreateExpenseInput) (model.Expense, error)
	CreateFamilyExpense(ctx context.Context, userID int64, input model.CreateExpenseInput) (model.Expense, error)

	ListExpenses(ctx context.Context, actor model.Actor, filter model.ExpenseFilter) (model.ExpensePage, error)
	ListFamilyExpenses(ctx context.Context, filter model.FamilyExpenseFilter) (model.ExpensePage, error)
	SummarizeExpenses(ctx context.Context, actor model.Actor, filter model.ExpenseFilter) (model.ExpenseSummary, error)
	LatestExpense(ctx context.Context, actor model.Actor) (model.Expense, error)
	LatestExpenseFiltered(ctx context.Context, actor model.Actor, filter model.ExpenseFilter) (model.Expense, error)

	GetExpense(ctx context.Context, actor model.Actor, id int64) (model.Expense, error)
	GetFamilyExpense(ctx context.Context, id int64) (model.Expense, error)

	UpdateExpense(ctx context.Context, actor model.Actor, id int64, patch model.UpdateExpenseInput) (model.Expense, error)
	UpdateFamilyExpense(ctx context.Context, id int64, patch model.UpdateExpenseInput) (model.Expense, error)

	DeleteExpense(ctx context.Context, actor model.Actor, id int64) error
	DeleteExpenseIfUnchanged(ctx context.Context, actor model.Actor, expense model.Expense) error
	DeleteFamilyExpense(ctx context.Context, id int64) error
}
