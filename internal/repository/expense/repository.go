package expense

import (
	"database/sql"

	"github.com/Saveliy12/expenses/internal/repository"
)

const expenseColumns = `
	id,
	user_id,
	amount,
	category,
	spent_at,
	COALESCE(comment, ''),
	created_at,
	updated_at
`

const expenseColumnsWithAlias = `
	e.id,
	e.user_id,
	e.amount,
	e.category,
	e.spent_at,
	COALESCE(e.comment, ''),
	e.created_at,
	e.updated_at
`

var _ repository.ExpenseRepository = (*Repository)(nil)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}
