package user

import (
	"context"
	"fmt"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/postgres"
)

func (r *Repository) ListFamilyMembers(ctx context.Context, familyID int64) ([]model.User, error) {
	const query = `
		SELECT ` + userColumns + `
		FROM users
		WHERE family_id = $1
		ORDER BY display_name, telegram_id
	`

	rows, err := r.db.QueryContext(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("list family users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		user, err := postgres.ScanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan family user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate family users: %w", err)
	}

	return users, nil
}
