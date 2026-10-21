package expense

import (
	"time"

	"github.com/Saveliy12/expenses/internal/repository"
	"github.com/Saveliy12/expenses/internal/service"
)

var _ service.ExpenseService = (*Service)(nil)

type Service struct {
	users    repository.UserRepository
	families repository.FamilyRepository
	expenses repository.ExpenseRepository
	now      func() time.Time
}

func NewService(
	users repository.UserRepository,
	families repository.FamilyRepository,
	expenses repository.ExpenseRepository,
) *Service {
	return &Service{
		users:    users,
		families: families,
		expenses: expenses,
		now:      time.Now,
	}
}
