package bot

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/Saveliy12/expenses/internal/model"
)

type fakeRegistrar struct {
	profile model.UserProfile
	calls   int
	err     error
}

func (f *fakeRegistrar) EnsureUser(_ context.Context, profile model.UserProfile) (model.User, error) {
	f.calls++
	f.profile = profile
	if f.err != nil {
		return model.User{}, f.err
	}

	return model.User{
		TelegramID:  profile.TelegramID,
		DisplayName: profile.DisplayName,
		Username:    profile.Username,
	}, nil
}

type fakeProcessor struct {
	actor model.Actor
	text  string
	reply string
	err   error
	calls int
}

func (f *fakeProcessor) Process(_ context.Context, actor model.Actor, text string) (string, error) {
	f.calls++
	f.actor = actor
	f.text = text
	return f.reply, f.err
}

func newTestHandler() (*Handler, *fakeRegistrar, *fakeProcessor) {
	registrar := &fakeRegistrar{}
	processor := &fakeProcessor{reply: "готово"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewHandler(registrar, processor, logger), registrar, processor
}

func TestHandleText(t *testing.T) {
	handler, registrar, processor := newTestHandler()

	reply := handler.Handle(context.Background(), Message{
		TelegramID: 42,
		Username:   "saveliy",
		FirstName:  "Савелий",
		LastName:   "Семенычев",
		Text:       "  кофе 350  ",
		IsPrivate:  true,
	})

	if reply != "готово" {
		t.Fatalf("expected processor reply, got %q", reply)
	}
	if registrar.calls != 1 {
		t.Fatalf("expected one EnsureUser call, got %d", registrar.calls)
	}
	if registrar.profile.DisplayName != "Савелий Семенычев" {
		t.Fatalf("unexpected display name %q", registrar.profile.DisplayName)
	}
	if registrar.profile.Username != "saveliy" {
		t.Fatalf("unexpected username %q", registrar.profile.Username)
	}
	if processor.calls != 1 {
		t.Fatalf("expected one processor call, got %d", processor.calls)
	}
	if processor.actor.TelegramID != 42 {
		t.Fatalf("unexpected actor %+v", processor.actor)
	}
	if processor.text != "кофе 350" {
		t.Fatalf("expected trimmed text, got %q", processor.text)
	}
}

func TestStartCommand(t *testing.T) {
	handler, registrar, processor := newTestHandler()

	reply := handler.Handle(context.Background(), Message{
		TelegramID: 42,
		FirstName:  "Савелий",
		Text:       "/start@expenses_bot",
		IsPrivate:  true,
	})

	if reply != Welcome {
		t.Fatalf("expected welcome message, got %q", reply)
	}
	if registrar.calls != 1 {
		t.Fatalf("expected user registration, got %d calls", registrar.calls)
	}
	if processor.calls != 0 {
		t.Fatalf("processor must not be called for /start, got %d calls", processor.calls)
	}
}

func TestUnknownCommand(t *testing.T) {
	handler, registrar, processor := newTestHandler()

	reply := handler.Handle(context.Background(), Message{
		TelegramID: 42,
		FirstName:  "Савелий",
		Text:       "/help",
		IsPrivate:  true,
	})

	if reply != UnknownCommand {
		t.Fatalf("expected unknown command message, got %q", reply)
	}
	if registrar.calls != 1 {
		t.Fatalf("expected user registration, got %d calls", registrar.calls)
	}
	if processor.calls != 0 {
		t.Fatalf("processor must not be called for unknown commands, got %d calls", processor.calls)
	}
}

func TestGroupChat(t *testing.T) {
	handler, registrar, processor := newTestHandler()

	reply := handler.Handle(context.Background(), Message{
		TelegramID: 42,
		Text:       "кофе 350",
		IsPrivate:  false,
	})

	if reply != OnlyPrivate {
		t.Fatalf("expected private chat message, got %q", reply)
	}
	if registrar.calls != 0 || processor.calls != 0 {
		t.Fatal("group message must not touch the service layer")
	}
}

func TestEmptyText(t *testing.T) {
	handler, registrar, processor := newTestHandler()

	reply := handler.Handle(context.Background(), Message{
		TelegramID: 42,
		Text:       "   ",
		IsPrivate:  true,
	})

	if reply != "" {
		t.Fatalf("expected no reply, got %q", reply)
	}
	if registrar.calls != 0 || processor.calls != 0 {
		t.Fatal("empty message must not touch the service layer")
	}
}

func TestEnsureUserError(t *testing.T) {
	handler, registrar, processor := newTestHandler()
	registrar.err = errors.New("database is down")

	reply := handler.Handle(context.Background(), Message{
		TelegramID: 42,
		Text:       "кофе 350",
		IsPrivate:  true,
	})

	if reply != ProcessingError {
		t.Fatalf("expected processing error, got %q", reply)
	}
	if processor.calls != 0 {
		t.Fatal("processor must not be called after registration failure")
	}
}

func TestProcessorError(t *testing.T) {
	handler, _, processor := newTestHandler()
	processor.err = errors.New("llm is down")

	reply := handler.Handle(context.Background(), Message{
		TelegramID: 42,
		Text:       "кофе 350",
		IsPrivate:  true,
	})

	if reply != ProcessingError {
		t.Fatalf("expected processing error, got %q", reply)
	}
}

func TestDisplayNameFallback(t *testing.T) {
	tests := []struct {
		name     string
		message  Message
		expected string
	}{
		{
			name:     "first and last name",
			message:  Message{FirstName: "Савелий", LastName: "Семенычев"},
			expected: "Савелий Семенычев",
		},
		{
			name:     "username",
			message:  Message{Username: "@saveliy"},
			expected: "saveliy",
		},
		{
			name:     "default",
			message:  Message{},
			expected: "Пользователь 42",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, registrar, _ := newTestHandler()
			test.message.TelegramID = 42
			test.message.Text = "кофе 350"
			test.message.IsPrivate = true

			handler.Handle(context.Background(), test.message)

			if registrar.profile.DisplayName != test.expected {
				t.Fatalf("expected display name %q, got %q", test.expected, registrar.profile.DisplayName)
			}
		})
	}
}
