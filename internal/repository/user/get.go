package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) GetByTelegramID(ctx context.Context, telegramID int64) (model.User, error) {
	const query = `
		SELECT ` + userColumns + `
		FROM users
		WHERE telegram_id = $1
	`

	user, err := postgres.ScanUser(r.db.QueryRowContext(ctx, query, telegramID))
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("get user by telegram ID: %w", err)
	}

	return user, nil
}
