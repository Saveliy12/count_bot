package bot

import (
	"context"
	"testing"

	"github.com/go-telegram/bot/models"
)

func TestNewRunnerValidation(t *testing.T) {
	t.Run("empty token", func(t *testing.T) {
		if _, err := NewRunner(context.Background(), "   ", nil, nil); err == nil {
			t.Fatal("expected an error for an empty token")
		}
	})

	t.Run("nil handler", func(t *testing.T) {
		if _, err := NewRunner(context.Background(), "token", nil, nil); err == nil {
			t.Fatal("expected an error for a nil handler")
		}
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if _, err := NewRunner(ctx, "token", &Handler{}, nil); err == nil {
			t.Fatal("expected an error for a canceled context")
		}
	})
}

func TestMessageFromUpdate(t *testing.T) {
	t.Run("private text message", func(t *testing.T) {
		update := &models.Update{
			Message: &models.Message{
				From: &models.User{
					ID:        42,
					FirstName: "Савелий",
					LastName:  "Семенычев",
					Username:  "saveliy",
				},
				Chat: models.Chat{ID: 42, Type: models.ChatTypePrivate},
				Text: "кофе 350",
			},
		}

		message, chatID, ok := messageFromUpdate(update)
		if !ok {
			t.Fatal("expected a message")
		}
		if chatID != 42 {
			t.Fatalf("expected chat ID 42, got %d", chatID)
		}
		if message.TelegramID != 42 || message.Text != "кофе 350" {
			t.Fatalf("unexpected message: %+v", message)
		}
		if !message.IsPrivate {
			t.Fatal("expected a private message")
		}
	})

	t.Run("group message", func(t *testing.T) {
		update := &models.Update{
			Message: &models.Message{
				From: &models.User{ID: 42, FirstName: "Савелий"},
				Chat: models.Chat{ID: -100, Type: models.ChatTypeGroup},
				Text: "кофе 350",
			},
		}

		message, _, ok := messageFromUpdate(update)
		if !ok {
			t.Fatal("expected a message")
		}
		if message.IsPrivate {
			t.Fatal("expected a group message")
		}
	})

	t.Run("missing message", func(t *testing.T) {
		if _, _, ok := messageFromUpdate(&models.Update{}); ok {
			t.Fatal("expected no message")
		}
	})

	t.Run("bot sender", func(t *testing.T) {
		update := &models.Update{
			Message: &models.Message{
				From: &models.User{ID: 42, IsBot: true},
				Chat: models.Chat{ID: 42, Type: models.ChatTypePrivate},
				Text: "кофе 350",
			},
		}

		if _, _, ok := messageFromUpdate(update); ok {
			t.Fatal("expected bot messages to be ignored")
		}
	})
}
