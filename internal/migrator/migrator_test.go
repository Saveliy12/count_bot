package migrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/Saveliy12/expenses/internal/migrator"
	"github.com/Saveliy12/expenses/internal/testdb"
)

func TestUp(t *testing.T) {
	db := testdb.NewSchema(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrator.Up(ctx, db); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if err := migrator.Up(ctx, db); err != nil {
		t.Fatalf("apply migrations second time: %v", err)
	}

	var tableCount int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*)
		FROM information_schema.tables
		WHERE table_schema = current_schema()
		  AND table_name IN ('families', 'users', 'expenses', 'goose_db_version')
	`).Scan(&tableCount); err != nil {
		t.Fatalf("count tables: %v", err)
	}
	if tableCount != 4 {
		t.Fatalf("expected 4 tables, got %d", tableCount)
	}

	var defaultFamilies int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM families WHERE is_default
	`).Scan(&defaultFamilies); err != nil {
		t.Fatalf("count default families: %v", err)
	}
	if defaultFamilies != 1 {
		t.Fatalf("expected one default family, got %d", defaultFamilies)
	}

	if _, err := db.ExecContext(ctx, `
		INSERT INTO families (name, is_default) VALUES ('Вторая семья', TRUE)
	`); err == nil {
		t.Fatal("expected unique default family constraint violation")
	}

	var familyID int64
	if err := db.QueryRowContext(ctx, `
		SELECT id FROM families WHERE is_default
	`).Scan(&familyID); err != nil {
		t.Fatalf("get default family ID: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO users (telegram_id, family_id, display_name)
		VALUES (1, $1, 'Тестовый пользователь')
	`, familyID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO expenses (user_id, amount, category, spent_at)
		VALUES (1, 100, 'personal_care', CURRENT_DATE)
	`); err != nil {
		t.Fatalf("insert expense with a new category: %v", err)
	}
}
