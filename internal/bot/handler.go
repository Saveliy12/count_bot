package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Saveliy12/expenses/internal/model"
)

type UserRegistrar interface {
	EnsureUser(ctx context.Context, profile model.UserProfile) (model.User, error)
}

type pendingResetter interface {
	ResetPending(telegramID int64)
}

type Message struct {
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
	Text       string
	IsPrivate  bool
}

type Handler struct {
	users     UserRegistrar
	processor TextProcessor
	logger    *slog.Logger
}

func NewHandler(users UserRegistrar, processor TextProcessor, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return &Handler{
		users:     users,
		processor: processor,
		logger:    logger,
	}
}

func (h *Handler) Handle(ctx context.Context, message Message) string {
	if !message.IsPrivate {
		return OnlyPrivate
	}

	text := strings.TrimSpace(message.Text)
	if text == "" {
		return ""
	}

	user, err := h.ensureUser(ctx, message)
	if err != nil {
		h.logger.Error("ensure telegram user", "error", err, "telegram_id", message.TelegramID)
		return ProcessingError
	}

	if isStartCommand(text) {
		h.resetPending(message.TelegramID)
		return Welcome
	}
	if strings.HasPrefix(text, "/") {
		h.resetPending(message.TelegramID)
		return UnknownCommand
	}

	reply, err := h.processor.Process(ctx, model.Actor{TelegramID: user.TelegramID}, text)
	if err != nil {
		h.logger.Error("process telegram message", "error", err, "telegram_id", message.TelegramID)
		return ProcessingError
	}

	return strings.TrimSpace(reply)
}

func (h *Handler) resetPending(telegramID int64) {
	if resetter, ok := h.processor.(pendingResetter); ok {
		resetter.ResetPending(telegramID)
	}
}

func (h *Handler) ensureUser(ctx context.Context, message Message) (model.User, error) {
	return h.users.EnsureUser(ctx, model.UserProfile{
		TelegramID:  message.TelegramID,
		DisplayName: displayName(message),
		Username:    message.Username,
	})
}

func displayName(message Message) string {
	name := strings.TrimSpace(strings.Join([]string{message.FirstName, message.LastName}, " "))
	if name != "" {
		return name
	}

	name = strings.TrimSpace(strings.TrimPrefix(message.Username, "@"))
	if name != "" {
		return name
	}

	return fmt.Sprintf("Пользователь %d", message.TelegramID)
}

func isStartCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}

	command := fields[0]
	if index := strings.IndexByte(command, '@'); index >= 0 {
		command = command[:index]
	}

	return strings.EqualFold(command, "/start")
}
