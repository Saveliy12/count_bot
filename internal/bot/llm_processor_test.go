package bot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/llm"
	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/service"
)

type fakeIntentProvider struct {
	intent  llm.Intent
	err     error
	request llm.Request
}

func (f *fakeIntentProvider) Parse(_ context.Context, req llm.Request) (llm.Intent, error) {
	f.request = req
	return f.intent, f.err
}

type scriptedProvider struct {
	intents  []llm.Intent
	requests []llm.Request
}

func (p *scriptedProvider) Parse(_ context.Context, req llm.Request) (llm.Intent, error) {
	p.requests = append(p.requests, req)
	if len(p.intents) == 0 {
		return llm.Intent{}, fmt.Errorf("no scripted intent")
	}

	intent := p.intents[0]
	p.intents = p.intents[1:]

	return intent, nil
}

type fakeExpenseService struct {
	createFn            func(context.Context, model.Actor, model.CreateExpenseInput) (model.Expense, error)
	listFn              func(context.Context, model.Actor, model.ExpenseFilter) (model.ExpensePage, error)
	summaryFn           func(context.Context, model.Actor, model.ExpenseFilter) (model.ExpenseSummary, error)
	latestFn            func(context.Context, model.Actor) (model.Expense, error)
	latestFilteredFn    func(context.Context, model.Actor, model.ExpenseFilter) (model.Expense, error)
	getFn               func(context.Context, model.Actor, int64) (model.Expense, error)
	updateFn            func(context.Context, model.Actor, int64, model.UpdateExpenseInput) (model.Expense, error)
	deleteFn            func(context.Context, model.Actor, int64) error
	deleteIfUnchangedFn func(context.Context, model.Actor, model.Expense) error
}

func (f *fakeExpenseService) EnsureUser(context.Context, model.UserProfile) (model.User, error) {
	return model.User{}, fmt.Errorf("unexpected EnsureUser call")
}

func (f *fakeExpenseService) ListFamilyUsers(context.Context) ([]model.User, error) {
	return nil, fmt.Errorf("unexpected ListFamilyUsers call")
}

func (f *fakeExpenseService) CreateFamilyExpense(
	context.Context,
	int64,
	model.CreateExpenseInput,
) (model.Expense, error) {
	return model.Expense{}, fmt.Errorf("unexpected CreateFamilyExpense call")
}

func (f *fakeExpenseService) ListFamilyExpenses(
	context.Context,
	model.FamilyExpenseFilter,
) (model.ExpensePage, error) {
	return model.ExpensePage{}, fmt.Errorf("unexpected ListFamilyExpenses call")
}

func (f *fakeExpenseService) GetFamilyExpense(context.Context, int64) (model.Expense, error) {
	return model.Expense{}, fmt.Errorf("unexpected GetFamilyExpense call")
}

func (f *fakeExpenseService) UpdateFamilyExpense(
	context.Context,
	int64,
	model.UpdateExpenseInput,
) (model.Expense, error) {
	return model.Expense{}, fmt.Errorf("unexpected UpdateFamilyExpense call")
}

func (f *fakeExpenseService) DeleteFamilyExpense(context.Context, int64) error {
	return fmt.Errorf("unexpected DeleteFamilyExpense call")
}

func (f *fakeExpenseService) CreateExpense(
	ctx context.Context,
	actor model.Actor,
	input model.CreateExpenseInput,
) (model.Expense, error) {
	if f.createFn == nil {
		return model.Expense{}, fmt.Errorf("unexpected CreateExpense call")
	}
	return f.createFn(ctx, actor, input)
}

func (f *fakeExpenseService) ListExpenses(
	ctx context.Context,
	actor model.Actor,
	filter model.ExpenseFilter,
) (model.ExpensePage, error) {
	if f.listFn == nil {
		return model.ExpensePage{}, fmt.Errorf("unexpected ListExpenses call")
	}
	return f.listFn(ctx, actor, filter)
}

func (f *fakeExpenseService) SummarizeExpenses(
	ctx context.Context,
	actor model.Actor,
	filter model.ExpenseFilter,
) (model.ExpenseSummary, error) {
	if f.summaryFn == nil {
		return model.ExpenseSummary{}, fmt.Errorf("unexpected SummarizeExpenses call")
	}
	return f.summaryFn(ctx, actor, filter)
}

func (f *fakeExpenseService) LatestExpense(
	ctx context.Context,
	actor model.Actor,
) (model.Expense, error) {
	if f.latestFn == nil {
		return model.Expense{}, fmt.Errorf("unexpected LatestExpense call")
	}
	return f.latestFn(ctx, actor)
}

func (f *fakeExpenseService) LatestExpenseFiltered(
	ctx context.Context,
	actor model.Actor,
	filter model.ExpenseFilter,
) (model.Expense, error) {
	if f.latestFilteredFn == nil {
		return model.Expense{}, fmt.Errorf("unexpected LatestExpenseFiltered call")
	}
	return f.latestFilteredFn(ctx, actor, filter)
}

func (f *fakeExpenseService) GetExpense(
	ctx context.Context,
	actor model.Actor,
	id int64,
) (model.Expense, error) {
	if f.getFn == nil {
		return model.Expense{}, fmt.Errorf("unexpected GetExpense call")
	}
	return f.getFn(ctx, actor, id)
}

func (f *fakeExpenseService) UpdateExpense(
	ctx context.Context,
	actor model.Actor,
	id int64,
	patch model.UpdateExpenseInput,
) (model.Expense, error) {
	if f.updateFn == nil {
		return model.Expense{}, fmt.Errorf("unexpected UpdateExpense call")
	}
	return f.updateFn(ctx, actor, id, patch)
}

func (f *fakeExpenseService) DeleteExpense(
	ctx context.Context,
	actor model.Actor,
	id int64,
) error {
	if f.deleteFn == nil {
		return fmt.Errorf("unexpected DeleteExpense call")
	}
	return f.deleteFn(ctx, actor, id)
}

func (f *fakeExpenseService) DeleteExpenseIfUnchanged(
	ctx context.Context,
	actor model.Actor,
	expense model.Expense,
) error {
	if f.deleteIfUnchangedFn == nil {
		return fmt.Errorf("unexpected DeleteExpenseIfUnchanged call")
	}
	return f.deleteIfUnchangedFn(ctx, actor, expense)
}

var _ service.ExpenseService = (*fakeExpenseService)(nil)

func newTestLLMProcessor(provider llm.Provider, expenses service.ExpenseService) *LLMProcessor {
	processor := NewLLMProcessor(
		provider,
		expenses,
		time.UTC,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	processor.now = func() time.Time {
		return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	}

	return processor
}

func TestLLMProcessorCreate(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:   llm.ActionCreate,
		Amount:   decimalPtr("350"),
		Category: categoryPtr(model.CategoryFood),
		DateHint: stringPtr("today"),
		Comment:  stringPtr("кофе"),
	}}

	var captured model.CreateExpenseInput
	expenses := &fakeExpenseService{
		createFn: func(_ context.Context, _ model.Actor, input model.CreateExpenseInput) (model.Expense, error) {
			captured = input
			return model.Expense{
				ID:       1,
				UserID:   42,
				Amount:   input.Amount,
				Category: input.Category,
				SpentAt:  input.SpentAt,
				Comment:  input.Comment,
			}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"кофе 350",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}

	if !captured.SpentAt.Equal(testDate(2026, 10, 21)) {
		t.Fatalf("unexpected spent date %s", captured.SpentAt)
	}
	if captured.Category != model.CategoryFood {
		t.Fatalf("unexpected category %q", captured.Category)
	}
	if !captured.Amount.Equal(decimal.NewFromInt(350)) {
		t.Fatalf("unexpected amount %s", captured.Amount)
	}
	if captured.Comment != "кофе" {
		t.Fatalf("unexpected comment %q", captured.Comment)
	}
	if !strings.Contains(reply, "Записал") || !strings.Contains(reply, "350") {
		t.Fatalf("unexpected reply %q", reply)
	}
	if provider.request.Today.Format(isoDateLayout) != "2026-10-21" {
		t.Fatalf("unexpected today passed to provider %s", provider.request.Today)
	}
}

func TestLLMProcessorCreateWithoutAmount(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{Action: llm.ActionCreate}}
	expenses := &fakeExpenseService{
		createFn: func(context.Context, model.Actor, model.CreateExpenseInput) (model.Expense, error) {
			t.Fatal("CreateExpense must not be called without an amount")
			return model.Expense{}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"потратил",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "сумм") {
		t.Fatalf("expected an amount clarification, got %q", reply)
	}
}

func TestLLMProcessorCreateWithoutCategory(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action: llm.ActionCreate,
		Amount: decimalPtr("350"),
	}}
	expenses := &fakeExpenseService{
		createFn: func(context.Context, model.Actor, model.CreateExpenseInput) (model.Expense, error) {
			t.Fatal("CreateExpense must not be called without a category")
			return model.Expense{}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"потратил 350",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "категории") {
		t.Fatalf("expected a category clarification, got %q", reply)
	}
}

func TestLLMProcessorClearComment(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:       llm.ActionUpdate,
		TargetMode:   llm.TargetLatest,
		ClearComment: true,
	}}
	expenses := &fakeExpenseService{
		latestFn: func(context.Context, model.Actor) (model.Expense, error) {
			return model.Expense{
				ID:       5,
				Amount:   decimal.NewFromInt(350),
				Category: model.CategoryFood,
				Comment:  "кофе",
			}, nil
		},
		updateFn: func(
			_ context.Context,
			_ model.Actor,
			_ int64,
			patch model.UpdateExpenseInput,
		) (model.Expense, error) {
			if patch.Comment == nil || *patch.Comment != "" {
				t.Fatalf("expected an empty comment patch, got %v", patch.Comment)
			}
			return model.Expense{
				ID:       5,
				Amount:   decimal.NewFromInt(350),
				Category: model.CategoryFood,
			}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"убери комментарий",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "Изменил") {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestCommentSuffixTruncates(t *testing.T) {
	short := commentSuffix("кофе")
	if short != " (кофе)" {
		t.Fatalf("unexpected short comment %q", short)
	}

	long := commentSuffix(strings.Repeat("а", displayedCommentLimit+20))
	if !strings.Contains(long, "…") {
		t.Fatalf("expected a truncated comment, got %q", long)
	}
	if len([]rune(long)) > displayedCommentLimit+3 {
		t.Fatalf("comment suffix is too long: %d runes", len([]rune(long)))
	}
}

func TestLLMProcessorCreateValidationError(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:   llm.ActionCreate,
		Amount:   decimalPtr("10.999"),
		Category: categoryPtr(model.CategoryFood),
	}}
	expenses := &fakeExpenseService{
		createFn: func(context.Context, model.Actor, model.CreateExpenseInput) (model.Expense, error) {
			return model.Expense{}, model.NewValidationError("amount", "must have at most two decimal places")
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"кофе 10.999",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "Сумма") {
		t.Fatalf("expected a friendly validation message, got %q", reply)
	}
}

func TestLLMProcessorList(t *testing.T) {
	category := model.CategoryFood
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:       llm.ActionList,
		Category:     &category,
		PeriodPreset: llm.PeriodMonth,
	}}

	var captured model.ExpenseFilter
	expenses := &fakeExpenseService{
		summaryFn: func(
			_ context.Context,
			_ model.Actor,
			filter model.ExpenseFilter,
		) (model.ExpenseSummary, error) {
			if filter.Category == nil || *filter.Category != model.CategoryFood {
				t.Fatalf("unexpected summary category filter %v", filter.Category)
			}
			return model.ExpenseSummary{Total: decimal.NewFromInt(500), Count: 2}, nil
		},
		listFn: func(
			_ context.Context,
			_ model.Actor,
			filter model.ExpenseFilter,
		) (model.ExpensePage, error) {
			captured = filter
			return model.ExpensePage{Items: []model.Expense{
				{ID: 1, Amount: decimal.NewFromInt(350), Category: model.CategoryFood, SpentAt: testDate(2026, 10, 20), Comment: "кофе"},
				{ID: 2, Amount: decimal.NewFromInt(150), Category: model.CategoryFood, SpentAt: testDate(2026, 10, 21), Comment: "обед"},
			}}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"сколько потратил на еду",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}

	if !captured.From.Equal(testDate(2026, 10, 1)) {
		t.Fatalf("unexpected from date %s", captured.From)
	}
	if !captured.To.Equal(testDate(2026, 10, 21)) {
		t.Fatalf("unexpected to date %s", captured.To)
	}
	if captured.Category == nil || *captured.Category != model.CategoryFood {
		t.Fatalf("unexpected category filter %v", captured.Category)
	}
	if captured.Limit != listItemLimit {
		t.Fatalf("unexpected limit %d", captured.Limit)
	}
	if !strings.Contains(reply, "500") || !strings.Contains(reply, "трат: 2") {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestLLMProcessorUpdateLatest(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:     llm.ActionUpdate,
		Amount:     decimalPtr("450"),
		TargetMode: llm.TargetLatest,
	}}
	expenses := &fakeExpenseService{
		latestFn: func(context.Context, model.Actor) (model.Expense, error) {
			return model.Expense{
				ID:       5,
				Amount:   decimal.NewFromInt(350),
				Category: model.CategoryFood,
				SpentAt:  testDate(2026, 10, 20),
			}, nil
		},
		updateFn: func(
			_ context.Context,
			_ model.Actor,
			id int64,
			patch model.UpdateExpenseInput,
		) (model.Expense, error) {
			if id != 5 {
				t.Fatalf("unexpected target ID %d", id)
			}
			if patch.Amount == nil || !patch.Amount.Equal(decimal.NewFromInt(450)) {
				t.Fatalf("unexpected amount patch %v", patch.Amount)
			}
			return model.Expense{
				ID:       id,
				Amount:   *patch.Amount,
				Category: model.CategoryFood,
				SpentAt:  testDate(2026, 10, 20),
			}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"последнее было не 350, а 450",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "Изменил") || !strings.Contains(reply, "450") {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestLLMProcessorDeleteByQuery(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:      llm.ActionDelete,
		TargetMode:  llm.TargetQuery,
		TargetQuery: stringPtr("такси"),
	}}
	expenses := &fakeExpenseService{
		listFn: func(
			_ context.Context,
			_ model.Actor,
			filter model.ExpenseFilter,
		) (model.ExpensePage, error) {
			if filter.Limit != targetSearchPageSize {
				t.Fatalf("unexpected search limit %d", filter.Limit)
			}
			return model.ExpensePage{Items: []model.Expense{
				{ID: 2, Amount: decimal.NewFromInt(1200), Category: model.CategoryTransport, SpentAt: testDate(2026, 10, 20), Comment: "такси"},
				{ID: 1, Amount: decimal.NewFromInt(350), Category: model.CategoryFood, SpentAt: testDate(2026, 10, 21), Comment: "кофе"},
			}}, nil
		},
		deleteIfUnchangedFn: func(_ context.Context, _ model.Actor, expense model.Expense) error {
			if expense.ID != 2 {
				t.Fatalf("unexpected deleted ID %d", expense.ID)
			}
			return nil
		},
	}

	processor := newTestLLMProcessor(provider, expenses)
	reply, err := processor.Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"удали такси",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "Удалить") || !strings.Contains(reply, "да") {
		t.Fatalf("expected a confirmation request, got %q", reply)
	}

	reply, err = processor.Process(context.Background(), model.Actor{TelegramID: 42}, "да")
	if err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
	if !strings.Contains(reply, "Удалил") || !strings.Contains(reply, "такси") {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestLLMProcessorDeleteConflict(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:     llm.ActionDelete,
		TargetMode: llm.TargetLatest,
	}}
	expenses := &fakeExpenseService{
		latestFn: func(context.Context, model.Actor) (model.Expense, error) {
			return model.Expense{ID: 5, Amount: decimal.NewFromInt(350), Category: model.CategoryFood}, nil
		},
		deleteIfUnchangedFn: func(context.Context, model.Actor, model.Expense) error {
			return model.ErrConflict
		},
	}

	processor := newTestLLMProcessor(provider, expenses)
	if _, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "удали последнее"); err != nil {
		t.Fatalf("request delete: %v", err)
	}

	reply, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "да")
	if err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
	if !strings.Contains(reply, "изменилась") {
		t.Fatalf("expected a conflict message, got %q", reply)
	}
}

func TestLLMProcessorDeleteCanceled(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:     llm.ActionDelete,
		TargetMode: llm.TargetLatest,
	}}
	expenses := &fakeExpenseService{
		latestFn: func(context.Context, model.Actor) (model.Expense, error) {
			return model.Expense{ID: 5, Amount: decimal.NewFromInt(350), Category: model.CategoryFood}, nil
		},
	}

	processor := newTestLLMProcessor(provider, expenses)
	if _, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "удали последнее"); err != nil {
		t.Fatalf("request delete: %v", err)
	}

	reply, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "нет")
	if err != nil {
		t.Fatalf("cancel delete: %v", err)
	}
	if !strings.Contains(reply, "Отменил") {
		t.Fatalf("expected cancellation, got %q", reply)
	}
}

func TestLLMProcessorTargetAmountIsExact(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:      llm.ActionDelete,
		TargetMode:  llm.TargetQuery,
		TargetQuery: stringPtr("350"),
	}}
	expenses := &fakeExpenseService{
		listFn: func(context.Context, model.Actor, model.ExpenseFilter) (model.ExpensePage, error) {
			return model.ExpensePage{Items: []model.Expense{
				{ID: 2, Amount: decimal.NewFromInt(1350), Category: model.CategoryOther, Comment: "такси"},
				{ID: 3, Amount: decimal.NewFromInt(350), Category: model.CategoryFood, Comment: "кофе"},
			}}, nil
		},
		deleteIfUnchangedFn: func(_ context.Context, _ model.Actor, expense model.Expense) error {
			if expense.ID != 3 {
				t.Fatalf("expected exact amount match ID 3, got %d", expense.ID)
			}
			return nil
		},
	}

	processor := newTestLLMProcessor(provider, expenses)
	reply, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "удали 350")
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if strings.Contains(reply, "несколько") {
		t.Fatalf("exact amount must not match 1350: %q", reply)
	}

	if _, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "да"); err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
}

func TestLLMProcessorTargetSearchContinuesAfterFirstPage(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:      llm.ActionDelete,
		TargetMode:  llm.TargetQuery,
		TargetQuery: stringPtr("такси"),
	}}
	expenses := &fakeExpenseService{
		listFn: func(
			_ context.Context,
			_ model.Actor,
			filter model.ExpenseFilter,
		) (model.ExpensePage, error) {
			if filter.Offset == 0 {
				return model.ExpensePage{
					Items:   []model.Expense{{ID: 1, Amount: decimal.NewFromInt(350), Comment: "кофе"}},
					HasMore: true,
				}, nil
			}
			return model.ExpensePage{
				Items: []model.Expense{{ID: 2, Amount: decimal.NewFromInt(1200), Comment: "такси"}},
			}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"удали такси",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "Удалить") {
		t.Fatalf("expected a confirmation request, got %q", reply)
	}
}

func TestLLMProcessorDeleteUsesTargetDateAndCategory(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:         llm.ActionDelete,
		TargetMode:     llm.TargetQuery,
		TargetDateHint: stringPtr("yesterday"),
		TargetCategory: categoryPtr(model.CategoryFood),
	}}
	expenses := &fakeExpenseService{
		listFn: func(
			_ context.Context,
			_ model.Actor,
			filter model.ExpenseFilter,
		) (model.ExpensePage, error) {
			if !filter.From.Equal(testDate(2026, 10, 20)) || !filter.To.Equal(testDate(2026, 10, 20)) {
				t.Fatalf("unexpected target date range: from=%s to=%s", filter.From, filter.To)
			}
			if filter.Category == nil || *filter.Category != model.CategoryFood {
				t.Fatalf("unexpected target category %v", filter.Category)
			}
			return model.ExpensePage{Items: []model.Expense{
				{ID: 4, Amount: decimal.NewFromInt(800), Category: model.CategoryFood, SpentAt: testDate(2026, 10, 20), Comment: "обед"},
			}}, nil
		},
		deleteIfUnchangedFn: func(_ context.Context, _ model.Actor, expense model.Expense) error {
			if expense.ID != 4 {
				t.Fatalf("unexpected deleted ID %d", expense.ID)
			}
			return nil
		},
	}

	processor := newTestLLMProcessor(provider, expenses)
	reply, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "удали вчерашние траты на еду")
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "Удалить") {
		t.Fatalf("expected a confirmation request, got %q", reply)
	}

	if _, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "да"); err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
}

func TestLLMProcessorUpdateLatestFiltered(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:         llm.ActionUpdate,
		Amount:         decimalPtr("500"),
		TargetMode:     llm.TargetLatest,
		TargetDateHint: stringPtr("yesterday"),
	}}
	expenses := &fakeExpenseService{
		latestFilteredFn: func(
			_ context.Context,
			_ model.Actor,
			filter model.ExpenseFilter,
		) (model.Expense, error) {
			if !filter.From.Equal(testDate(2026, 10, 20)) || !filter.To.Equal(testDate(2026, 10, 20)) {
				t.Fatalf("unexpected latest filter: from=%s to=%s", filter.From, filter.To)
			}
			return model.Expense{
				ID:       5,
				Amount:   decimal.NewFromInt(350),
				Category: model.CategoryFood,
				SpentAt:  testDate(2026, 10, 20),
			}, nil
		},
		updateFn: func(
			_ context.Context,
			_ model.Actor,
			id int64,
			patch model.UpdateExpenseInput,
		) (model.Expense, error) {
			if id != 5 {
				t.Fatalf("unexpected target ID %d", id)
			}
			if patch.Amount == nil || !patch.Amount.Equal(decimal.NewFromInt(500)) {
				t.Fatalf("unexpected amount patch %v", patch.Amount)
			}
			return model.Expense{
				ID:       5,
				Amount:   *patch.Amount,
				Category: model.CategoryFood,
				SpentAt:  testDate(2026, 10, 20),
			}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"измени последнюю вчерашнюю трату на 500",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "Изменил") || !strings.Contains(reply, "500") {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestLLMProcessorUpdateOldAmountMismatch(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:     llm.ActionUpdate,
		Amount:     decimalPtr("450"),
		OldAmount:  decimalPtr("350"),
		TargetMode: llm.TargetLatest,
	}}
	expenses := &fakeExpenseService{
		latestFn: func(context.Context, model.Actor) (model.Expense, error) {
			return model.Expense{ID: 5, Amount: decimal.NewFromInt(100), Category: model.CategoryFood}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"последнее было не 350, а 450",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "100") || !strings.Contains(reply, "350") {
		t.Fatalf("expected an old amount mismatch message, got %q", reply)
	}
}

func TestLLMProcessorDeleteAmbiguous(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:      llm.ActionDelete,
		TargetMode:  llm.TargetQuery,
		TargetQuery: stringPtr("такси"),
	}}
	expenses := &fakeExpenseService{
		listFn: func(context.Context, model.Actor, model.ExpenseFilter) (model.ExpensePage, error) {
			return model.ExpensePage{Items: []model.Expense{
				{ID: 1, Amount: decimal.NewFromInt(1200), Category: model.CategoryTransport, Comment: "такси"},
				{ID: 2, Amount: decimal.NewFromInt(1300), Category: model.CategoryTransport, Comment: "такси"},
			}}, nil
		},
	}

	reply, err := newTestLLMProcessor(provider, expenses).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"удали такси",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "несколько") {
		t.Fatalf("expected an ambiguity clarification, got %q", reply)
	}
}

func TestLLMProcessorUpdateWithoutTarget(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action: llm.ActionUpdate,
		Amount: decimalPtr("450"),
	}}

	reply, err := newTestLLMProcessor(provider, &fakeExpenseService{}).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"измени на 450",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "какую трату") {
		t.Fatalf("expected a target clarification, got %q", reply)
	}
}

func TestLLMProcessorClarify(t *testing.T) {
	provider := &fakeIntentProvider{intent: llm.Intent{
		Action:        llm.ActionClarify,
		Clarification: "Сколько и на что была трата?",
	}}

	reply, err := newTestLLMProcessor(provider, &fakeExpenseService{}).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"запиши трату",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if reply != "Сколько и на что была трата?" {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestLLMProcessorClarificationContext(t *testing.T) {
	provider := &scriptedProvider{intents: []llm.Intent{
		{Action: llm.ActionClarify, Clarification: "Сколько и на что была трата?"},
		{
			Action:   llm.ActionCreate,
			Amount:   decimalPtr("350"),
			Category: categoryPtr(model.CategoryFood),
			DateHint: stringPtr("today"),
			Comment:  stringPtr("кофе"),
		},
	}}

	var captured model.CreateExpenseInput
	expenses := &fakeExpenseService{
		createFn: func(_ context.Context, _ model.Actor, input model.CreateExpenseInput) (model.Expense, error) {
			captured = input
			return model.Expense{
				ID:       1,
				Amount:   input.Amount,
				Category: input.Category,
				SpentAt:  input.SpentAt,
				Comment:  input.Comment,
			}, nil
		},
	}
	processor := newTestLLMProcessor(provider, expenses)

	reply, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "запиши трату")
	if err != nil {
		t.Fatalf("first message: %v", err)
	}
	if !strings.Contains(reply, "Сколько") {
		t.Fatalf("unexpected clarification %q", reply)
	}

	reply, err = processor.Process(context.Background(), model.Actor{TelegramID: 42}, "350 на кофе")
	if err != nil {
		t.Fatalf("second message: %v", err)
	}
	if !strings.Contains(reply, "Записал") {
		t.Fatalf("expected a created expense, got %q", reply)
	}
	if !captured.Amount.Equal(decimal.NewFromInt(350)) {
		t.Fatalf("unexpected amount %s", captured.Amount)
	}
	if len(provider.requests) != 2 {
		t.Fatalf("expected 2 LLM requests, got %d", len(provider.requests))
	}
	contextText := provider.requests[1].Text
	if !strings.Contains(contextText, "Предыдущее сообщение") ||
		!strings.Contains(contextText, "Сколько и на что была трата?") ||
		!strings.Contains(contextText, "350 на кофе") {
		t.Fatalf("clarification context was not passed to the model: %q", contextText)
	}
}

func TestLLMProcessorClarificationExpires(t *testing.T) {
	provider := &scriptedProvider{intents: []llm.Intent{
		{Action: llm.ActionClarify, Clarification: "Сколько?"},
		{Action: llm.ActionClarify, Clarification: "Не понял, повтори."},
	}}
	processor := newTestLLMProcessor(provider, &fakeExpenseService{})

	current := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	processor.now = func() time.Time { return current }

	if _, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "запиши трату"); err != nil {
		t.Fatalf("first message: %v", err)
	}

	current = current.Add(pendingClarificationTTL + time.Minute)
	if _, err := processor.Process(context.Background(), model.Actor{TelegramID: 42}, "350"); err != nil {
		t.Fatalf("second message: %v", err)
	}

	if len(provider.requests) != 2 {
		t.Fatalf("expected 2 LLM requests, got %d", len(provider.requests))
	}
	if strings.Contains(provider.requests[1].Text, "Предыдущее сообщение") {
		t.Fatalf("expired clarification must not be passed to the model: %q", provider.requests[1].Text)
	}
}

func TestLLMProcessorRateLimited(t *testing.T) {
	provider := &fakeIntentProvider{err: llm.ErrRateLimited}

	reply, err := newTestLLMProcessor(provider, &fakeExpenseService{}).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"кофе 350",
	)
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	if !strings.Contains(reply, "много запросов") {
		t.Fatalf("unexpected reply %q", reply)
	}
}

func TestLLMProcessorProviderError(t *testing.T) {
	provider := &fakeIntentProvider{err: errors.New("network failure")}

	_, err := newTestLLMProcessor(provider, &fakeExpenseService{}).Process(
		context.Background(),
		model.Actor{TelegramID: 42},
		"кофе 350",
	)
	if err == nil {
		t.Fatal("expected an error from the provider")
	}
}

func testDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func decimalPtr(value string) *decimal.Decimal {
	amount := decimal.RequireFromString(value)
	return &amount
}

func categoryPtr(value model.Category) *model.Category {
	return &value
}

func stringPtr(value string) *string {
	return &value
}
