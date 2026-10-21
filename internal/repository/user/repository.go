package user

import (
	"database/sql"

	"github.com/Saveliy12/expenses/internal/repository"
)

const userColumns = `
	telegram_id,
	family_id,
	display_name,
	COALESCE(username, ''),
	created_at,
	updated_at
`

var _ repository.UserRepository = (*Repository)(nil)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}
