package family

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

var _ repository.FamilyRepository = (*Repository)(nil)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetDefault(ctx context.Context) (model.Family, error) {
	const query = `
		SELECT id, name, is_default, created_at
		FROM families
		WHERE is_default
		ORDER BY id
		LIMIT 1
	`

	family, err := postgres.ScanFamily(r.db.QueryRowContext(ctx, query))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Family{}, model.ErrNotFound
	}
	if err != nil {
		return model.Family{}, fmt.Errorf("get default family: %w", err)
	}

	return family, nil
}
