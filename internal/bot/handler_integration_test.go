package bot

import (
	"context"
	"io"
	"log/slog"
	"testing"

	repositoryexpense "github.com/Saveliy12/expenses/internal/repository/expense"
	"github.com/Saveliy12/expenses/internal/repository/family"
	"github.com/Saveliy12/expenses/internal/repository/user"
	serviceexpense "github.com/Saveliy12/expenses/internal/service/expense"
	"github.com/Saveliy12/expenses/internal/testdb"
)

func TestHandleRegistersUserInDatabase(t *testing.T) {
	db := testdb.New(t)

	expenseService := serviceexpense.NewService(
		user.NewRepository(db),
		family.NewRepository(db),
		repositoryexpense.NewRepository(db),
	)
	handler := NewHandler(
		expenseService,
		StubProcessor{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	ctx := context.Background()

	reply := handler.Handle(ctx, Message{
		TelegramID: 555001,
		FirstName:  "Савелий",
		Text:       "/start",
		IsPrivate:  true,
	})
	if reply != Welcome {
		t.Fatalf("expected welcome message, got %q", reply)
	}

	users, err := expenseService.ListFamilyUsers(ctx)
	if err != nil {
		t.Fatalf("list family users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected one registered user, got %d", len(users))
	}
	if users[0].TelegramID != 555001 {
		t.Fatalf("unexpected registered user: %+v", users[0])
	}
	if users[0].DisplayName != "Савелий" {
		t.Fatalf("unexpected display name %q", users[0].DisplayName)
	}

	reply = handler.Handle(ctx, Message{
		TelegramID: 555001,
		FirstName:  "Савелий",
		Text:       "кофе 350",
		IsPrivate:  true,
	})
	if reply != LLMNotReady {
		t.Fatalf("expected the LLM placeholder reply, got %q", reply)
	}
}
