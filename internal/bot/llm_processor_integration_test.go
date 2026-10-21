package bot

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/llm"
	"github.com/Saveliy12/expenses/internal/model"
	repositoryexpense "github.com/Saveliy12/expenses/internal/repository/expense"
	"github.com/Saveliy12/expenses/internal/repository/family"
	"github.com/Saveliy12/expenses/internal/repository/user"
	serviceexpense "github.com/Saveliy12/expenses/internal/service/expense"
	"github.com/Saveliy12/expenses/internal/testdb"
)

func TestLLMProcessorWithGroq(t *testing.T) {
	if os.Getenv("LLM_INTEGRATION") != "1" {
		t.Skip("set LLM_INTEGRATION=1 to run the real LLM integration test")
	}

	apiKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	if apiKey == "" {
		t.Skip("GROQ_API_KEY is not set")
	}

	db := testdb.New(t)
	expenseService := serviceexpense.NewService(
		user.NewRepository(db),
		family.NewRepository(db),
		repositoryexpense.NewRepository(db),
	)
	ctx := context.Background()

	modelName := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if modelName == "" {
		modelName = "openai/gpt-oss-120b"
	}
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	client := llm.NewClient(llm.Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
		Timeout: 20 * time.Second,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	processor := NewLLMProcessor(
		client,
		expenseService,
		time.UTC,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	processor.now = func() time.Time {
		return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	}

	if _, err := expenseService.EnsureUser(ctx, model.UserProfile{
		TelegramID:  555777,
		DisplayName: "Интеграционный тест",
	}); err != nil {
		t.Fatalf("ensure user: %v", err)
	}
	actor := model.Actor{TelegramID: 555777}

	reply, err := processor.Process(ctx, actor, "кофе 350")
	if err != nil {
		t.Fatalf("create via llm: %v", err)
	}
	if !strings.Contains(reply, "Записал") {
		t.Fatalf("unexpected create reply %q", reply)
	}

	page, err := expenseService.ListExpenses(ctx, actor, model.ExpenseFilter{})
	if err != nil {
		t.Fatalf("list expenses: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one expense, got %d", len(page.Items))
	}
	created := page.Items[0]
	if !created.Amount.Equal(decimal.NewFromInt(350)) {
		t.Fatalf("unexpected amount %s", created.Amount)
	}
	if created.Category != model.CategoryFood {
		t.Fatalf("unexpected category %q", created.Category)
	}
	if created.SpentAt.Format(isoDateLayout) != "2026-10-21" {
		t.Fatalf("unexpected spent date %s", created.SpentAt)
	}

	reply, err = processor.Process(ctx, actor, "сколько потратил на еду")
	if err != nil {
		t.Fatalf("list via llm: %v", err)
	}
	if !strings.Contains(reply, "350") || !strings.Contains(reply, "трат: 1") {
		t.Fatalf("unexpected list reply %q", reply)
	}

	reply, err = processor.Process(ctx, actor, "последнее было не 350, а 450")
	if err != nil {
		t.Fatalf("update via llm: %v", err)
	}
	if !strings.Contains(reply, "450") {
		t.Fatalf("unexpected update reply %q", reply)
	}

	latest, err := expenseService.LatestExpense(ctx, actor)
	if err != nil {
		t.Fatalf("latest expense: %v", err)
	}
	if !latest.Amount.Equal(decimal.NewFromInt(450)) {
		t.Fatalf("expected updated amount 450, got %s", latest.Amount)
	}

	reply, err = processor.Process(ctx, actor, "удали последнее")
	if err != nil {
		t.Fatalf("delete via llm: %v", err)
	}
	if !strings.Contains(reply, "Удалить") {
		t.Fatalf("expected a delete confirmation, got %q", reply)
	}

	reply, err = processor.Process(ctx, actor, "да")
	if err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
	if !strings.Contains(reply, "Удалил") {
		t.Fatalf("unexpected delete reply %q", reply)
	}

	page, err = expenseService.ListExpenses(ctx, actor, model.ExpenseFilter{})
	if err != nil {
		t.Fatalf("list expenses after delete: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected no expenses after delete, got %d", len(page.Items))
	}
}
