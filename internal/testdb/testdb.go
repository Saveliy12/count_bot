package testdb

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/Saveliy12/expenses/internal/migrator"
)

func New(tb testing.TB) *sql.DB {
	tb.Helper()

	db := NewSchema(tb)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := migrator.Up(ctx, db); err != nil {
		tb.Fatalf("apply test migrations: %v", err)
	}

	return db
}

func NewSchema(tb testing.TB) *sql.DB {
	tb.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		tb.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		tb.Fatalf("open admin database connection: %v", err)
	}
	if err := admin.PingContext(ctx); err != nil {
		_ = admin.Close()
		tb.Fatalf("ping admin database connection: %v", err)
	}

	schema := "test_" + randomSuffix(tb)
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		_ = admin.Close()
		tb.Fatalf("create test schema: %v", err)
	}

	tb.Cleanup(func() {
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dropCancel()
		_, _ = admin.ExecContext(dropCtx, "DROP SCHEMA IF EXISTS "+quotedSchema+" CASCADE")
		_ = admin.Close()
	})

	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		tb.Fatalf("parse test database URL: %v", err)
	}
	config.RuntimeParams["search_path"] = schema

	db := stdlib.OpenDB(*config)
	tb.Cleanup(func() {
		_ = db.Close()
	})

	if err := db.PingContext(ctx); err != nil {
		tb.Fatalf("ping test database: %v", err)
	}

	return db
}

func randomSuffix(tb testing.TB) string {
	tb.Helper()

	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	return hex.EncodeToString(random)
}
