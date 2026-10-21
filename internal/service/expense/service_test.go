package expense

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

var fixedNow = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

type testClock struct {
	current time.Time
}

func (c *testClock) Now() time.Time {
	return c.current
}

type fakeStore struct {
	family        model.Family
	users         map[int64]model.User
	expenses      map[int64]model.Expense
	nextExpenseID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		family: model.Family{
			ID:        7,
			Name:      "Моя семья",
			IsDefault: true,
		},
		users:         make(map[int64]model.User),
		expenses:      make(map[int64]model.Expense),
		nextExpenseID: 1,
	}
}

func (s *fakeStore) GetDefault(context.Context) (model.Family, error) {
	return s.family, nil
}

func (s *fakeStore) Upsert(_ context.Context, user model.User) (model.User, error) {
	if existing, ok := s.users[user.TelegramID]; ok {
		existing.DisplayName = user.DisplayName
		existing.Username = user.Username
		existing.UpdatedAt = user.UpdatedAt
		s.users[user.TelegramID] = existing
		return existing, nil
	}

	s.users[user.TelegramID] = user
	return user, nil
}

func (s *fakeStore) GetByTelegramID(_ context.Context, telegramID int64) (model.User, error) {
	user, ok := s.users[telegramID]
	if !ok {
		return model.User{}, model.ErrNotFound
	}
	return user, nil
}

func (s *fakeStore) ListFamilyMembers(_ context.Context, familyID int64) ([]model.User, error) {
	var result []model.User
	for _, user := range s.users {
		if user.FamilyID == familyID {
			result = append(result, user)
		}
	}

	slices.SortFunc(result, func(a, b model.User) int {
		switch {
		case a.DisplayName < b.DisplayName:
			return -1
		case a.DisplayName > b.DisplayName:
			return 1
		default:
			return 0
		}
	})

	return result, nil
}

func (s *fakeStore) Create(_ context.Context, expense model.Expense) (model.Expense, error) {
	if _, ok := s.users[expense.UserID]; !ok {
		return model.Expense{}, model.ErrNotFound
	}

	expense.ID = s.nextExpenseID
	s.nextExpenseID++
	s.expenses[expense.ID] = expense
	return expense, nil
}

func (s *fakeStore) GetByID(_ context.Context, id int64) (model.Expense, error) {
	expense, ok := s.expenses[id]
	if !ok {
		return model.Expense{}, model.ErrNotFound
	}
	return expense, nil
}

func (s *fakeStore) ListByUser(
	_ context.Context,
	userID int64,
	filter model.ExpenseFilter,
) (model.ExpensePage, error) {
	var result []model.Expense
	for _, expense := range s.expenses {
		if expense.UserID != userID {
			continue
		}
		if filter.Category != nil && expense.Category != *filter.Category {
			continue
		}
		if !filter.From.IsZero() && expense.SpentAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && expense.SpentAt.After(filter.To) {
			continue
		}
		result = append(result, expense)
	}

	slices.SortFunc(result, func(a, b model.Expense) int {
		switch {
		case a.SpentAt.After(b.SpentAt):
			return -1
		case a.SpentAt.Before(b.SpentAt):
			return 1
		case a.ID > b.ID:
			return -1
		case a.ID < b.ID:
			return 1
		default:
			return 0
		}
	})

	if filter.Offset >= len(result) {
		return model.ExpensePage{}, nil
	}
	result = result[filter.Offset:]

	page := model.ExpensePage{}
	if filter.Limit > 0 && len(result) > filter.Limit {
		page.HasMore = true
		result = result[:filter.Limit]
	}
	page.Items = result

	return page, nil
}

func (s *fakeStore) ListByFamily(
	_ context.Context,
	familyID int64,
	filter model.FamilyExpenseFilter,
) (model.ExpensePage, error) {
	var result []model.Expense
	for _, expense := range s.expenses {
		user, ok := s.users[expense.UserID]
		if !ok || user.FamilyID != familyID {
			continue
		}
		if filter.UserID > 0 && expense.UserID != filter.UserID {
			continue
		}
		if filter.Category != nil && expense.Category != *filter.Category {
			continue
		}
		if !filter.From.IsZero() && expense.SpentAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && expense.SpentAt.After(filter.To) {
			continue
		}
		result = append(result, expense)
	}

	slices.SortFunc(result, func(a, b model.Expense) int {
		switch {
		case a.SpentAt.After(b.SpentAt):
			return -1
		case a.SpentAt.Before(b.SpentAt):
			return 1
		case a.ID > b.ID:
			return -1
		case a.ID < b.ID:
			return 1
		default:
			return 0
		}
	})

	if filter.Offset >= len(result) {
		return model.ExpensePage{}, nil
	}
	result = result[filter.Offset:]

	page := model.ExpensePage{}
	if filter.Limit > 0 && len(result) > filter.Limit {
		page.HasMore = true
		result = result[:filter.Limit]
	}
	page.Items = result

	return page, nil
}

func (s *fakeStore) SummaryByUser(
	_ context.Context,
	userID int64,
	filter model.ExpenseFilter,
) (model.ExpenseSummary, error) {
	total := decimal.Zero
	var count int64
	for _, expense := range s.expenses {
		if expense.UserID != userID {
			continue
		}
		if filter.Category != nil && expense.Category != *filter.Category {
			continue
		}
		if !filter.From.IsZero() && expense.SpentAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && expense.SpentAt.After(filter.To) {
			continue
		}
		total = total.Add(expense.Amount)
		count++
	}

	return model.ExpenseSummary{Total: total, Count: count}, nil
}

func (s *fakeStore) LatestByUser(ctx context.Context, userID int64) (model.Expense, error) {
	return s.LatestByUserFiltered(ctx, userID, model.ExpenseFilter{})
}

func (s *fakeStore) LatestByUserFiltered(
	_ context.Context,
	userID int64,
	filter model.ExpenseFilter,
) (model.Expense, error) {
	var (
		latest model.Expense
		found  bool
	)
	for _, expense := range s.expenses {
		if expense.UserID != userID {
			continue
		}
		if filter.Category != nil && expense.Category != *filter.Category {
			continue
		}
		if !filter.From.IsZero() && expense.SpentAt.Before(filter.From) {
			continue
		}
		if !filter.To.IsZero() && expense.SpentAt.After(filter.To) {
			continue
		}
		if !found ||
			expense.CreatedAt.After(latest.CreatedAt) ||
			(expense.CreatedAt.Equal(latest.CreatedAt) && expense.ID > latest.ID) {
			latest = expense
			found = true
		}
	}
	if !found {
		return model.Expense{}, model.ErrNotFound
	}

	return latest, nil
}

func (s *fakeStore) Update(_ context.Context, expense model.Expense) (model.Expense, error) {
	if _, ok := s.expenses[expense.ID]; !ok {
		return model.Expense{}, model.ErrNotFound
	}
	s.expenses[expense.ID] = expense
	return expense, nil
}

func (s *fakeStore) Delete(_ context.Context, id int64) error {
	if _, ok := s.expenses[id]; !ok {
		return model.ErrNotFound
	}
	delete(s.expenses, id)
	return nil
}

func (s *fakeStore) DeleteIfUnchanged(_ context.Context, expense model.Expense) error {
	existing, ok := s.expenses[expense.ID]
	if !ok {
		return model.ErrConflict
	}
	if existing.UserID != expense.UserID ||
		!existing.Amount.Equal(expense.Amount) ||
		existing.Category != expense.Category ||
		!existing.SpentAt.Equal(expense.SpentAt) ||
		existing.Comment != expense.Comment ||
		!existing.UpdatedAt.Equal(expense.UpdatedAt) {
		return model.ErrConflict
	}

	delete(s.expenses, expense.ID)
	return nil
}

func newTestService() (*Service, *fakeStore, *testClock) {
	store := newFakeStore()
	clock := &testClock{current: fixedNow}
	service := NewService(store, store, store)
	service.now = clock.Now
	return service, store, clock
}

func TestEnsureUser(t *testing.T) {
	service, _, clock := newTestService()
	ctx := context.Background()

	user, err := service.EnsureUser(ctx, model.UserProfile{
		TelegramID:  42,
		DisplayName: "  Савелий  ",
		Username:    " @saveliy ",
	})
	if err != nil {
		t.Fatalf("ensure user: %v", err)
	}
	if user.FamilyID != 7 {
		t.Fatalf("expected family ID 7, got %d", user.FamilyID)
	}
	if user.DisplayName != "Савелий" {
		t.Fatalf("expected trimmed display name, got %q", user.DisplayName)
	}
	if user.Username != "saveliy" {
		t.Fatalf("expected username without @, got %q", user.Username)
	}
	if !user.CreatedAt.Equal(fixedNow) || !user.UpdatedAt.Equal(fixedNow) {
		t.Fatalf("unexpected timestamps: created=%s updated=%s", user.CreatedAt, user.UpdatedAt)
	}

	clock.current = fixedNow.Add(time.Hour)
	updated, err := service.EnsureUser(ctx, model.UserProfile{
		TelegramID:  42,
		DisplayName: "Савелий Новый",
		Username:    "saveliy_new",
	})
	if err != nil {
		t.Fatalf("update user profile: %v", err)
	}
	if !updated.CreatedAt.Equal(fixedNow) {
		t.Fatalf("expected original creation time, got %s", updated.CreatedAt)
	}
	if !updated.UpdatedAt.Equal(fixedNow.Add(time.Hour)) {
		t.Fatalf("expected updated timestamp, got %s", updated.UpdatedAt)
	}
	if updated.FamilyID != 7 {
		t.Fatalf("expected family to stay unchanged, got %d", updated.FamilyID)
	}

	_, err = service.EnsureUser(ctx, model.UserProfile{TelegramID: 42, DisplayName: "   "})
	assertValidationField(t, err, "display_name")
}

func TestCreateExpense(t *testing.T) {
	service, store, _ := newTestService()
	ctx := context.Background()
	actor := mustEnsureUser(t, service, 42, "Савелий")

	expense, err := service.CreateExpense(ctx, actor, model.CreateExpenseInput{
		Amount:   decimal.RequireFromString("350.50"),
		Category: model.CategoryFood,
		SpentAt:  time.Date(2026, 10, 19, 18, 30, 0, 0, time.UTC),
		Comment:  "  кофе  ",
	})
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}
	if expense.ID != 1 {
		t.Fatalf("expected expense ID 1, got %d", expense.ID)
	}
	if expense.UserID != actor.TelegramID {
		t.Fatalf("expected owner %d, got %d", actor.TelegramID, expense.UserID)
	}
	if !expense.SpentAt.Equal(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("expected date without time, got %s", expense.SpentAt)
	}
	if expense.Comment != "кофе" {
		t.Fatalf("expected trimmed comment, got %q", expense.Comment)
	}
	if len(store.expenses) != 1 {
		t.Fatalf("expected one stored expense, got %d", len(store.expenses))
	}

	_, err = service.CreateExpense(ctx, model.Actor{TelegramID: 999}, model.CreateExpenseInput{
		Amount:   decimal.NewFromInt(100),
		Category: model.CategoryOther,
		SpentAt:  fixedNow,
	})
	if !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found for unknown user, got %v", err)
	}

	_, err = service.CreateExpense(ctx, actor, model.CreateExpenseInput{
		Amount:   decimal.RequireFromString("10.999"),
		Category: model.CategoryFood,
		SpentAt:  fixedNow,
	})
	assertValidationField(t, err, "amount")
}

func TestExpenseOwnership(t *testing.T) {
	service, _, _ := newTestService()
	ctx := context.Background()
	owner := mustEnsureUser(t, service, 42, "Савелий")
	other := mustEnsureUser(t, service, 43, "Другой")
	expense := mustCreateExpense(t, service, owner, 100, model.CategoryFood, "2026-10-19")

	if _, err := service.GetExpense(ctx, other, expense.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found for another user, got %v", err)
	}

	amount := decimal.NewFromInt(200)
	if _, err := service.UpdateExpense(ctx, other, expense.ID, model.UpdateExpenseInput{
		Amount: &amount,
	}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found when updating another user's expense, got %v", err)
	}

	if err := service.DeleteExpense(ctx, other, expense.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found when deleting another user's expense, got %v", err)
	}
}

func TestUpdateAndDeleteExpense(t *testing.T) {
	service, store, clock := newTestService()
	ctx := context.Background()
	actor := mustEnsureUser(t, service, 42, "Савелий")
	expense := mustCreateExpense(t, service, actor, 100, model.CategoryFood, "2026-10-19")

	clock.current = fixedNow.Add(time.Hour)
	amount := decimal.NewFromInt(450)
	category := model.CategoryTransport
	comment := "  метро  "
	updated, err := service.UpdateExpense(ctx, actor, expense.ID, model.UpdateExpenseInput{
		Amount:   &amount,
		Category: &category,
		Comment:  &comment,
	})
	if err != nil {
		t.Fatalf("update expense: %v", err)
	}
	if !updated.Amount.Equal(amount) {
		t.Fatalf("expected amount %s, got %s", amount, updated.Amount)
	}
	if updated.Category != category {
		t.Fatalf("expected category %q, got %q", category, updated.Category)
	}
	if updated.Comment != "метро" {
		t.Fatalf("expected trimmed comment, got %q", updated.Comment)
	}
	if !updated.CreatedAt.Equal(expense.CreatedAt) {
		t.Fatalf("creation time changed")
	}
	if !updated.UpdatedAt.Equal(fixedNow.Add(time.Hour)) {
		t.Fatalf("unexpected update time %s", updated.UpdatedAt)
	}
	if !updated.SpentAt.Equal(expense.SpentAt) {
		t.Fatalf("spent date changed unexpectedly")
	}

	if _, err := service.UpdateExpense(ctx, actor, expense.ID, model.UpdateExpenseInput{}); !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected validation error for an empty patch, got %v", err)
	}

	invalidAmount := decimal.RequireFromString("10.999")
	if _, err := service.UpdateExpense(ctx, actor, expense.ID, model.UpdateExpenseInput{
		Amount: &invalidAmount,
	}); !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected validation error for invalid amount, got %v", err)
	}
	if stored := store.expenses[expense.ID]; !stored.Amount.Equal(amount) {
		t.Fatalf("invalid update was stored: %s", stored.Amount)
	}

	if err := service.DeleteExpense(ctx, actor, expense.ID); err != nil {
		t.Fatalf("delete expense: %v", err)
	}
	if _, err := service.GetExpense(ctx, actor, expense.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected deleted expense to be missing, got %v", err)
	}
}

func TestListExpensesAndLatest(t *testing.T) {
	service, _, _ := newTestService()
	ctx := context.Background()
	actor := mustEnsureUser(t, service, 42, "Савелий")
	other := mustEnsureUser(t, service, 43, "Другой")

	food := mustCreateExpense(t, service, actor, 100, model.CategoryFood, "2026-10-18")
	transport := mustCreateExpense(t, service, actor, 200, model.CategoryTransport, "2026-10-19")
	mustCreateExpense(t, service, actor, 300, model.CategoryFood, "2026-10-20")
	mustCreateExpense(t, service, other, 50, model.CategoryOther, "2026-10-21")

	expenses, err := service.ListExpenses(ctx, actor, model.ExpenseFilter{})
	if err != nil {
		t.Fatalf("list expenses: %v", err)
	}
	if len(expenses.Items) != 3 {
		t.Fatalf("expected 3 expenses, got %d", len(expenses.Items))
	}
	if expenses.Items[0].SpentAt.Format("2006-01-02") != "2026-10-20" {
		t.Fatalf("expected newest expense first, got %s", expenses.Items[0].SpentAt)
	}
	if expenses.HasMore {
		t.Fatal("expected no more pages with the default limit")
	}

	foodExpenses, err := service.ListExpenses(ctx, actor, model.ExpenseFilter{Category: &food.Category})
	if err != nil {
		t.Fatalf("filter by category: %v", err)
	}
	if len(foodExpenses.Items) != 2 {
		t.Fatalf("expected 2 food expenses, got %d", len(foodExpenses.Items))
	}

	day := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	dayExpenses, err := service.ListExpenses(ctx, actor, model.ExpenseFilter{From: day, To: day})
	if err != nil {
		t.Fatalf("filter by date: %v", err)
	}
	if len(dayExpenses.Items) != 1 || dayExpenses.Items[0].ID != transport.ID {
		t.Fatalf("expected only the transport expense, got %+v", dayExpenses.Items)
	}

	limited, err := service.ListExpenses(ctx, actor, model.ExpenseFilter{Limit: 1})
	if err != nil {
		t.Fatalf("list with limit: %v", err)
	}
	if len(limited.Items) != 1 {
		t.Fatalf("expected one expense, got %d", len(limited.Items))
	}
	if !limited.HasMore {
		t.Fatal("expected another page after the first of three expenses")
	}

	summary, err := service.SummarizeExpenses(ctx, actor, model.ExpenseFilter{})
	if err != nil {
		t.Fatalf("summarize expenses: %v", err)
	}
	if summary.Count != 3 || !summary.Total.Equal(decimal.NewFromInt(600)) {
		t.Fatalf("unexpected summary: count=%d total=%s", summary.Count, summary.Total)
	}

	foodSummary, err := service.SummarizeExpenses(ctx, actor, model.ExpenseFilter{Category: &food.Category})
	if err != nil {
		t.Fatalf("summarize food expenses: %v", err)
	}
	if foodSummary.Count != 2 || !foodSummary.Total.Equal(decimal.NewFromInt(400)) {
		t.Fatalf("unexpected food summary: count=%d total=%s", foodSummary.Count, foodSummary.Total)
	}

	latest, err := service.LatestExpense(ctx, actor)
	if err != nil {
		t.Fatalf("latest expense: %v", err)
	}
	if latest.SpentAt.Format("2006-01-02") != "2026-10-20" {
		t.Fatalf("unexpected latest expense date %s", latest.SpentAt)
	}

	filteredLatest, err := service.LatestExpenseFiltered(ctx, actor, model.ExpenseFilter{
		Category: &food.Category,
	})
	if err != nil {
		t.Fatalf("latest filtered expense: %v", err)
	}
	if filteredLatest.ID != 3 {
		t.Fatalf("expected filtered latest ID 3, got %d", filteredLatest.ID)
	}

	if _, err := service.LatestExpense(ctx, model.Actor{TelegramID: 999}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found when there are no expenses, got %v", err)
	}
}

func TestDeleteExpenseIfUnchanged(t *testing.T) {
	service, store, clock := newTestService()
	ctx := context.Background()
	actor := mustEnsureUser(t, service, 42, "Савелий")
	expense := mustCreateExpense(t, service, actor, 100, model.CategoryFood, "2026-10-19")

	clock.current = fixedNow.Add(time.Hour)
	amount := decimal.NewFromInt(450)
	updated, err := service.UpdateExpense(ctx, actor, expense.ID, model.UpdateExpenseInput{
		Amount: &amount,
	})
	if err != nil {
		t.Fatalf("update expense: %v", err)
	}

	if err := service.DeleteExpenseIfUnchanged(ctx, actor, expense); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("expected a stale snapshot conflict, got %v", err)
	}
	if _, ok := store.expenses[expense.ID]; !ok {
		t.Fatal("stale delete must not remove the expense")
	}

	if err := service.DeleteExpenseIfUnchanged(ctx, actor, updated); err != nil {
		t.Fatalf("delete updated snapshot: %v", err)
	}
	if _, ok := store.expenses[expense.ID]; ok {
		t.Fatal("expected the expense to be deleted")
	}

	other := mustEnsureUser(t, service, 43, "Другой")
	otherExpense := mustCreateExpense(t, service, other, 200, model.CategoryOther, "2026-10-19")
	if err := service.DeleteExpenseIfUnchanged(ctx, actor, otherExpense); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found for another user's snapshot, got %v", err)
	}
}

func TestFamilyExpenses(t *testing.T) {
	service, store, _ := newTestService()
	ctx := context.Background()
	userA := mustEnsureUser(t, service, 42, "Савелий")
	userB := mustEnsureUser(t, service, 43, "Другой")

	familyUsers, err := service.ListFamilyUsers(ctx)
	if err != nil {
		t.Fatalf("list family users: %v", err)
	}
	if len(familyUsers) != 2 {
		t.Fatalf("expected 2 family users, got %d", len(familyUsers))
	}

	mustCreateExpense(t, service, userA, 100, model.CategoryFood, "2026-10-18")
	expenseB, err := service.CreateFamilyExpense(ctx, userB.TelegramID, model.CreateExpenseInput{
		Amount:   decimal.NewFromInt(200),
		Category: model.CategoryTravel,
		SpentAt:  time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		Comment:  "билет",
	})
	if err != nil {
		t.Fatalf("create family expense: %v", err)
	}

	familyExpenses, err := service.ListFamilyExpenses(ctx, model.FamilyExpenseFilter{})
	if err != nil {
		t.Fatalf("list family expenses: %v", err)
	}
	if len(familyExpenses.Items) != 2 {
		t.Fatalf("expected 2 family expenses, got %d", len(familyExpenses.Items))
	}

	userAFilter := model.FamilyExpenseFilter{UserID: userA.TelegramID}
	userAExpenses, err := service.ListFamilyExpenses(ctx, userAFilter)
	if err != nil {
		t.Fatalf("filter family expenses by user: %v", err)
	}
	if len(userAExpenses.Items) != 1 {
		t.Fatalf("expected one expense for user A, got %d", len(userAExpenses.Items))
	}

	if _, err := service.GetFamilyExpense(ctx, expenseB.ID); err != nil {
		t.Fatalf("get family expense: %v", err)
	}

	amount := decimal.NewFromInt(250)
	updated, err := service.UpdateFamilyExpense(ctx, expenseB.ID, model.UpdateExpenseInput{Amount: &amount})
	if err != nil {
		t.Fatalf("update family expense: %v", err)
	}
	if !updated.Amount.Equal(amount) {
		t.Fatalf("expected amount %s, got %s", amount, updated.Amount)
	}

	if err := service.DeleteFamilyExpense(ctx, expenseB.ID); err != nil {
		t.Fatalf("delete family expense: %v", err)
	}
	if _, err := service.GetFamilyExpense(ctx, expenseB.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected deleted family expense to be missing, got %v", err)
	}

	store.users[99] = model.User{
		TelegramID:  99,
		FamilyID:    99,
		DisplayName: "Чужой",
		CreatedAt:   fixedNow,
		UpdatedAt:   fixedNow,
	}
	otherExpense := mustCreateExpense(t, service, model.Actor{TelegramID: 99}, 300, model.CategoryOther, "2026-10-20")
	if _, err := service.GetFamilyExpense(ctx, otherExpense.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected another family's expense to be hidden, got %v", err)
	}
	if _, err := service.CreateFamilyExpense(ctx, 99, model.CreateExpenseInput{
		Amount:   decimal.NewFromInt(100),
		Category: model.CategoryOther,
		SpentAt:  fixedNow,
	}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected another family's user to be hidden, got %v", err)
	}

	familyExpenses, err = service.ListFamilyExpenses(ctx, model.FamilyExpenseFilter{})
	if err != nil {
		t.Fatalf("list family expenses after cross-family setup: %v", err)
	}
	if len(familyExpenses.Items) != 1 {
		t.Fatalf("expected only the default family's expense, got %d", len(familyExpenses.Items))
	}
}

func TestNormalizeFamilyFilter(t *testing.T) {
	from := time.Date(2026, 10, 18, 15, 30, 0, 0, time.UTC)
	to := time.Date(2026, 10, 20, 18, 45, 0, 0, time.UTC)
	category := model.CategoryFood

	filter, err := normalizeFamilyFilter(model.FamilyExpenseFilter{
		UserID:   42,
		From:     from,
		To:       to,
		Category: &category,
		Limit:    1000,
		Offset:   5,
	})
	if err != nil {
		t.Fatalf("normalize family filter: %v", err)
	}
	if filter.UserID != 42 {
		t.Fatalf("expected user ID 42, got %d", filter.UserID)
	}
	if filter.Limit != model.MaxPageSize {
		t.Fatalf("expected limit %d, got %d", model.MaxPageSize, filter.Limit)
	}
	if !filter.From.Equal(time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected normalized from date %s", filter.From)
	}
	if _, err := normalizeFamilyFilter(model.FamilyExpenseFilter{UserID: -1}); !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected validation error for negative user ID, got %v", err)
	}
}

func TestNormalizeFilter(t *testing.T) {
	from := time.Date(2026, 10, 18, 15, 30, 0, 0, time.UTC)
	to := time.Date(2026, 10, 20, 18, 45, 0, 0, time.UTC)
	category := model.CategoryFood

	filter, err := normalizeFilter(model.ExpenseFilter{
		From:     from,
		To:       to,
		Category: &category,
		Limit:    1000,
		Offset:   5,
	})
	if err != nil {
		t.Fatalf("normalize filter: %v", err)
	}
	if filter.Limit != model.MaxPageSize {
		t.Fatalf("expected limit %d, got %d", model.MaxPageSize, filter.Limit)
	}
	if filter.Offset != 5 {
		t.Fatalf("expected offset 5, got %d", filter.Offset)
	}
	if !filter.From.Equal(time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected normalized from date %s", filter.From)
	}
	if !filter.To.Equal(time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected normalized to date %s", filter.To)
	}

	defaults, err := normalizeFilter(model.ExpenseFilter{})
	if err != nil {
		t.Fatalf("normalize empty filter: %v", err)
	}
	if defaults.Limit != model.DefaultPageSize || defaults.Offset != 0 {
		t.Fatalf("unexpected defaults: limit=%d offset=%d", defaults.Limit, defaults.Offset)
	}

	invalid := model.Category("crypto")
	tests := []struct {
		name   string
		filter model.ExpenseFilter
	}{
		{name: "negative limit", filter: model.ExpenseFilter{Limit: -1}},
		{name: "negative offset", filter: model.ExpenseFilter{Offset: -1}},
		{name: "invalid category", filter: model.ExpenseFilter{Category: &invalid}},
		{
			name: "from after to",
			filter: model.ExpenseFilter{
				From: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := normalizeFilter(test.filter); !errors.Is(err, model.ErrValidation) {
				t.Fatalf("expected validation error, got %v", err)
			}
		})
	}
}

func mustEnsureUser(t *testing.T, service *Service, telegramID int64, name string) model.Actor {
	t.Helper()

	user, err := service.EnsureUser(context.Background(), model.UserProfile{
		TelegramID:  telegramID,
		DisplayName: name,
	})
	if err != nil {
		t.Fatalf("ensure user %d: %v", telegramID, err)
	}
	return model.Actor{TelegramID: user.TelegramID}
}

func mustCreateExpense(
	t *testing.T,
	service *Service,
	actor model.Actor,
	amount int64,
	category model.Category,
	date string,
) model.Expense {
	t.Helper()

	spentAt, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatalf("parse test date: %v", err)
	}

	expense, err := service.CreateExpense(context.Background(), actor, model.CreateExpenseInput{
		Amount:   decimal.NewFromInt(amount),
		Category: category,
		SpentAt:  spentAt,
	})
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}
	return expense
}

func assertValidationField(t *testing.T, err error, wantField string) {
	t.Helper()

	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}

	var validationErr *model.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if validationErr.Field != wantField {
		t.Fatalf("expected field %q, got %q", wantField, validationErr.Field)
	}
}
