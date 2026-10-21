package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	initAttempts = 3
	initBackoff  = 2 * time.Second
)

type Runner struct {
	api     *tgbot.Bot
	handler *Handler
	logger  *slog.Logger
}

func NewRunner(ctx context.Context, token string, handler *Handler, logger *slog.Logger) (*Runner, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("telegram bot token is empty")
	}
	if handler == nil {
		return nil, errors.New("telegram handler is nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}

	runner := &Runner{
		handler: handler,
		logger:  logger,
	}

	var (
		api *tgbot.Bot
		err error
	)
	for attempt := 1; attempt <= initAttempts; attempt++ {
		api, err = tgbot.New(
			token,
			tgbot.WithDefaultHandler(runner.handleUpdate),
			tgbot.WithWorkers(1),
		)
		if err == nil {
			break
		}
		if errors.Is(err, tgbot.ErrorUnauthorized) {
			return nil, fmt.Errorf("create telegram bot: %w", err)
		}
		if attempt == initAttempts {
			break
		}

		logger.Warn("telegram bot initialization failed, retrying",
			"attempt", attempt,
			"error", err,
		)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("create telegram bot: %w", ctx.Err())
		case <-time.After(initBackoff):
		}
	}
	if err != nil {
		return nil, fmt.Errorf("create telegram bot: %w", err)
	}
	runner.api = api

	return runner, nil
}

func (r *Runner) Start(ctx context.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.logger.Error("telegram bot panicked",
				"panic", recovered,
				"stack", string(debug.Stack()),
			)
		}
	}()

	r.logger.Info("telegram bot started")
	r.api.Start(ctx)
	r.logger.Info("telegram bot stopped")
}

func (r *Runner) handleUpdate(ctx context.Context, api *tgbot.Bot, update *models.Update) {
	defer func() {
		if recovered := recover(); recovered != nil {
			r.logger.Error("panic in telegram update",
				"panic", recovered,
				"stack", string(debug.Stack()),
			)
		}
	}()

	message, chatID, ok := messageFromUpdate(update)
	if !ok {
		return
	}

	if message.Text == "" {
		reply := OnlyText
		if !message.IsPrivate {
			reply = OnlyPrivate
		}
		r.sendMessage(ctx, api, chatID, reply)
		return
	}

	reply := r.handler.Handle(ctx, message)
	if reply == "" || ctx.Err() != nil {
		return
	}

	r.sendMessage(ctx, api, chatID, reply)
}

func (r *Runner) sendMessage(ctx context.Context, api *tgbot.Bot, chatID int64, text string) {
	if ctx.Err() != nil {
		return
	}

	if _, err := api.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	}); err != nil {
		r.logger.Error("send telegram message", "error", err, "chat_id", chatID)
	}
}

func messageFromUpdate(update *models.Update) (Message, int64, bool) {
	if update.Message == nil || update.Message.From == nil || update.Message.From.IsBot {
		return Message{}, 0, false
	}

	message := update.Message
	return Message{
		TelegramID: message.From.ID,
		Username:   message.From.Username,
		FirstName:  message.From.FirstName,
		LastName:   message.From.LastName,
		Text:       message.Text,
		IsPrivate:  message.Chat.Type == models.ChatTypePrivate,
	}, message.Chat.ID, true
}
