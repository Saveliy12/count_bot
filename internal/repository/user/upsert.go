package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) Upsert(ctx context.Context, user model.User) (model.User, error) {
	const query = `
		INSERT INTO users (
			telegram_id,
			family_id,
			display_name,
			username,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6)
		ON CONFLICT (telegram_id) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    username = EXCLUDED.username,
		    updated_at = EXCLUDED.updated_at
		RETURNING ` + userColumns

	result, err := postgres.ScanUser(r.db.QueryRowContext(
		ctx,
		query,
		user.TelegramID,
		user.FamilyID,
		user.DisplayName,
		user.Username,
		user.CreatedAt,
		user.UpdatedAt,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrNotFound
	}
	if err != nil {
		return model.User{}, fmt.Errorf("upsert user: %w", err)
	}

	return result, nil
}
