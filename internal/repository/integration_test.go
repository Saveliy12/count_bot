package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/repository/expense"
	"github.com/Saveliy12/expenses/internal/repository/family"
	"github.com/Saveliy12/expenses/internal/repository/user"
	"github.com/Saveliy12/expenses/internal/testdb"
)

var repoTestNow = time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)

func TestRepositories(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	families := family.NewRepository(db)
	users := user.NewRepository(db)
	expenses := expense.NewRepository(db)

	family, err := families.GetDefault(ctx)
	if err != nil {
		t.Fatalf("get default family: %v", err)
	}
	if !family.IsDefault {
		t.Fatal("expected the seeded family to be the default")
	}

	userA, err := users.Upsert(ctx, model.User{
		TelegramID:  42,
		FamilyID:    family.ID,
		DisplayName: "Савелий",
		Username:    "saveliy",
		CreatedAt:   repoTestNow,
		UpdatedAt:   repoTestNow,
	})
	if err != nil {
		t.Fatalf("create user A: %v", err)
	}
	if userA.TelegramID != 42 || userA.Username != "saveliy" {
		t.Fatalf("unexpected user A: %+v", userA)
	}

	userA, err = users.Upsert(ctx, model.User{
		TelegramID:  42,
		FamilyID:    family.ID,
		DisplayName: "Савелий Новый",
		Username:    "",
		CreatedAt:   repoTestNow.Add(-time.Hour),
		UpdatedAt:   repoTestNow.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("update user A: %v", err)
	}
	if userA.DisplayName != "Савелий Новый" {
		t.Fatalf("expected updated display name, got %q", userA.DisplayName)
	}
	if userA.Username != "" {
		t.Fatalf("expected empty username, got %q", userA.Username)
	}
	if !userA.CreatedAt.Equal(repoTestNow) {
		t.Fatalf("creation time changed: %s", userA.CreatedAt)
	}
	if !userA.UpdatedAt.Equal(repoTestNow.Add(time.Hour)) {
		t.Fatalf("unexpected update time: %s", userA.UpdatedAt)
	}

	storedUser, err := users.GetByTelegramID(ctx, 42)
	if err != nil {
		t.Fatalf("get user A: %v", err)
	}
	if storedUser.DisplayName != userA.DisplayName {
		t.Fatalf("unexpected stored user: %+v", storedUser)
	}

	userB, err := users.Upsert(ctx, model.User{
		TelegramID:  43,
		FamilyID:    family.ID,
		DisplayName: "Другой",
		CreatedAt:   repoTestNow,
		UpdatedAt:   repoTestNow,
	})
	if err != nil {
		t.Fatalf("create user B: %v", err)
	}

	members, err := users.ListFamilyMembers(ctx, family.ID)
	if err != nil {
		t.Fatalf("list family members: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 family members, got %d", len(members))
	}

	food := mustCreateExpense(t, expenses, model.Expense{
		UserID:    userA.TelegramID,
		Amount:    decimal.RequireFromString("100.50"),
		Category:  model.CategoryFood,
		SpentAt:   testDate(2026, 10, 18),
		Comment:   "продукты",
		CreatedAt: repoTestNow,
		UpdatedAt: repoTestNow,
	})
	transport := mustCreateExpense(t, expenses, model.Expense{
		UserID:    userA.TelegramID,
		Amount:    decimal.NewFromInt(200),
		Category:  model.CategoryTransport,
		SpentAt:   testDate(2026, 10, 19),
		CreatedAt: repoTestNow,
		UpdatedAt: repoTestNow,
	})
	food300 := mustCreateExpense(t, expenses, model.Expense{
		UserID:    userA.TelegramID,
		Amount:    decimal.NewFromInt(300),
		Category:  model.CategoryFood,
		SpentAt:   testDate(2026, 10, 20),
		CreatedAt: repoTestNow,
		UpdatedAt: repoTestNow,
	})
	mustCreateExpense(t, expenses, model.Expense{
		UserID:    userB.TelegramID,
		Amount:    decimal.NewFromInt(50),
		Category:  model.CategoryOther,
		SpentAt:   testDate(2026, 10, 21),
		CreatedAt: repoTestNow,
		UpdatedAt: repoTestNow,
	})

	familyList, err := expenses.ListByFamily(ctx, family.ID, model.FamilyExpenseFilter{})
	if err != nil {
		t.Fatalf("list family expenses: %v", err)
	}
	if len(familyList.Items) != 4 {
		t.Fatalf("expected 4 family expenses, got %d", len(familyList.Items))
	}
	if familyList.Items[0].SpentAt.Format("2006-01-02") != "2026-10-21" {
		t.Fatalf("expected newest family expense first, got %s", familyList.Items[0].SpentAt)
	}
	if familyList.HasMore {
		t.Fatal("expected no more family pages with the default limit")
	}

	familyForUserA, err := expenses.ListByFamily(ctx, family.ID, model.FamilyExpenseFilter{
		UserID: userA.TelegramID,
	})
	if err != nil {
		t.Fatalf("list family expenses by user: %v", err)
	}
	if len(familyForUserA.Items) != 3 {
		t.Fatalf("expected 3 expenses for user A, got %d", len(familyForUserA.Items))
	}

	var otherFamilyID int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO families (name, is_default) VALUES ('Другая семья', FALSE)
		RETURNING id
	`).Scan(&otherFamilyID); err != nil {
		t.Fatalf("create another family: %v", err)
	}
	userC, err := users.Upsert(ctx, model.User{
		TelegramID:  44,
		FamilyID:    otherFamilyID,
		DisplayName: "Из другой семьи",
		CreatedAt:   repoTestNow,
		UpdatedAt:   repoTestNow,
	})
	if err != nil {
		t.Fatalf("create user C: %v", err)
	}
	mustCreateExpense(t, expenses, model.Expense{
		UserID:    userC.TelegramID,
		Amount:    decimal.NewFromInt(999),
		Category:  model.CategoryOther,
		SpentAt:   testDate(2026, 10, 22),
		CreatedAt: repoTestNow,
		UpdatedAt: repoTestNow,
	})

	familyList, err = expenses.ListByFamily(ctx, family.ID, model.FamilyExpenseFilter{})
	if err != nil {
		t.Fatalf("list family expenses after adding another family: %v", err)
	}
	if len(familyList.Items) != 4 {
		t.Fatalf("another family's expense leaked into the list: %d", len(familyList.Items))
	}

	list, err := expenses.ListByUser(ctx, userA.TelegramID, model.ExpenseFilter{})
	if err != nil {
		t.Fatalf("list expenses: %v", err)
	}
	if len(list.Items) != 3 {
		t.Fatalf("expected 3 expenses for user A, got %d", len(list.Items))
	}
	if got := list.Items[0].SpentAt.Format("2006-01-02"); got != "2026-10-20" {
		t.Fatalf("expected newest expense first, got %s", got)
	}
	if !food.Amount.Equal(decimal.RequireFromString("100.50")) {
		t.Fatalf("unexpected decimal amount: %s", food.Amount)
	}

	foodCategory := model.CategoryFood
	foodList, err := expenses.ListByUser(ctx, userA.TelegramID, model.ExpenseFilter{
		Category: &foodCategory,
	})
	if err != nil {
		t.Fatalf("list food expenses: %v", err)
	}
	if len(foodList.Items) != 2 {
		t.Fatalf("expected 2 food expenses, got %d", len(foodList.Items))
	}

	summary, err := expenses.SummaryByUser(ctx, userA.TelegramID, model.ExpenseFilter{})
	if err != nil {
		t.Fatalf("summarize expenses: %v", err)
	}
	if summary.Count != 3 || !summary.Total.Equal(decimal.RequireFromString("600.50")) {
		t.Fatalf("unexpected summary: count=%d total=%s", summary.Count, summary.Total)
	}

	foodSummary, err := expenses.SummaryByUser(ctx, userA.TelegramID, model.ExpenseFilter{
		Category: &foodCategory,
	})
	if err != nil {
		t.Fatalf("summarize food expenses: %v", err)
	}
	if foodSummary.Count != 2 || !foodSummary.Total.Equal(decimal.RequireFromString("400.50")) {
		t.Fatalf("unexpected food summary: count=%d total=%s", foodSummary.Count, foodSummary.Total)
	}

	filteredLatest, err := expenses.LatestByUserFiltered(ctx, userA.TelegramID, model.ExpenseFilter{
		From:     testDate(2026, 10, 20),
		To:       testDate(2026, 10, 20),
		Category: &foodCategory,
	})
	if err != nil {
		t.Fatalf("get latest filtered expense: %v", err)
	}
	if filteredLatest.ID != food300.ID {
		t.Fatalf("expected filtered latest ID %d, got %d", food300.ID, filteredLatest.ID)
	}

	day := testDate(2026, 10, 19)
	dayList, err := expenses.ListByUser(ctx, userA.TelegramID, model.ExpenseFilter{
		From: day,
		To:   day,
	})
	if err != nil {
		t.Fatalf("list expenses for one day: %v", err)
	}
	if len(dayList.Items) != 1 || dayList.Items[0].ID != transport.ID {
		t.Fatalf("expected only the transport expense, got %+v", dayList.Items)
	}

	page, err := expenses.ListByUser(ctx, userA.TelegramID, model.ExpenseFilter{
		Limit:  1,
		Offset: 1,
	})
	if err != nil {
		t.Fatalf("list expenses with pagination: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != transport.ID {
		t.Fatalf("unexpected paginated expenses: %+v", page.Items)
	}
	if !page.HasMore {
		t.Fatal("expected another page after the first of the remaining expenses")
	}

	storedExpense, err := expenses.GetByID(ctx, transport.ID)
	if err != nil {
		t.Fatalf("get expense: %v", err)
	}
	if storedExpense.UserID != userA.TelegramID {
		t.Fatalf("unexpected expense owner: %d", storedExpense.UserID)
	}
	if storedExpense.Comment != "" {
		t.Fatalf("expected empty comment, got %q", storedExpense.Comment)
	}

	updated, err := expenses.Update(ctx, model.Expense{
		ID:        transport.ID,
		UserID:    userA.TelegramID,
		Amount:    decimal.NewFromInt(250),
		Category:  model.CategoryPersonalCare,
		SpentAt:   testDate(2026, 10, 22),
		Comment:   "стрижка",
		UpdatedAt: repoTestNow.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("update expense: %v", err)
	}
	if updated.Category != model.CategoryPersonalCare {
		t.Fatalf("expected updated category, got %q", updated.Category)
	}
	if updated.Comment != "стрижка" {
		t.Fatalf("expected updated comment, got %q", updated.Comment)
	}
	if updated.SpentAt.Format("2006-01-02") != "2026-10-22" {
		t.Fatalf("expected updated date, got %s", updated.SpentAt)
	}

	if _, err := expenses.Update(ctx, model.Expense{
		ID:        transport.ID,
		UserID:    userB.TelegramID,
		Amount:    decimal.NewFromInt(1),
		Category:  model.CategoryOther,
		SpentAt:   testDate(2026, 10, 22),
		UpdatedAt: repoTestNow,
	}); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found for another user's update, got %v", err)
	}

	if err := expenses.Delete(ctx, transport.ID); err != nil {
		t.Fatalf("delete expense: %v", err)
	}
	if _, err := expenses.GetByID(ctx, transport.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected deleted expense to be missing, got %v", err)
	}
	if err := expenses.Delete(ctx, transport.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected not found when deleting twice, got %v", err)
	}

	if _, err := families.GetDefault(ctx); err != nil {
		t.Fatalf("get default family after operations: %v", err)
	}
}

func TestLatestByUserUsesCreatedAt(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	families := family.NewRepository(db)
	users := user.NewRepository(db)
	expenses := expense.NewRepository(db)

	defaultFamily, err := families.GetDefault(ctx)
	if err != nil {
		t.Fatalf("get default family: %v", err)
	}
	if _, err := users.Upsert(ctx, model.User{
		TelegramID:  77,
		FamilyID:    defaultFamily.ID,
		DisplayName: "Тест",
		CreatedAt:   repoTestNow,
		UpdatedAt:   repoTestNow,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	olderCreatedNewerSpent := mustCreateExpense(t, expenses, model.Expense{
		UserID:    77,
		Amount:    decimal.NewFromInt(100),
		Category:  model.CategoryFood,
		SpentAt:   testDate(2026, 10, 20),
		CreatedAt: repoTestNow.Add(-time.Hour),
		UpdatedAt: repoTestNow.Add(-time.Hour),
	})
	newerCreatedOlderSpent := mustCreateExpense(t, expenses, model.Expense{
		UserID:    77,
		Amount:    decimal.NewFromInt(200),
		Category:  model.CategoryOther,
		SpentAt:   testDate(2026, 1, 1),
		CreatedAt: repoTestNow,
		UpdatedAt: repoTestNow,
	})

	latest, err := expenses.LatestByUser(ctx, 77)
	if err != nil {
		t.Fatalf("get latest expense: %v", err)
	}
	if latest.ID != newerCreatedOlderSpent.ID {
		t.Fatalf("expected latest created expense %d, got %d", newerCreatedOlderSpent.ID, latest.ID)
	}
	if latest.ID == olderCreatedNewerSpent.ID {
		t.Fatal("latest expense must not be selected by spent_at")
	}
}

func TestDeleteIfUnchanged(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()

	families := family.NewRepository(db)
	users := user.NewRepository(db)
	expenses := expense.NewRepository(db)

	defaultFamily, err := families.GetDefault(ctx)
	if err != nil {
		t.Fatalf("get default family: %v", err)
	}
	if _, err := users.Upsert(ctx, model.User{
		TelegramID:  88,
		FamilyID:    defaultFamily.ID,
		DisplayName: "Тест",
		CreatedAt:   repoTestNow,
		UpdatedAt:   repoTestNow,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	snapshot := mustCreateExpense(t, expenses, model.Expense{
		UserID:    88,
		Amount:    decimal.NewFromInt(100),
		Category:  model.CategoryFood,
		SpentAt:   testDate(2026, 10, 19),
		Comment:   "старое",
		CreatedAt: repoTestNow,
		UpdatedAt: repoTestNow,
	})

	updated, err := expenses.Update(ctx, model.Expense{
		ID:        snapshot.ID,
		UserID:    88,
		Amount:    decimal.NewFromInt(200),
		Category:  model.CategoryOther,
		SpentAt:   testDate(2026, 10, 20),
		Comment:   "новое",
		UpdatedAt: repoTestNow.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("update expense: %v", err)
	}

	if err := expenses.DeleteIfUnchanged(ctx, snapshot); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("expected a stale snapshot conflict, got %v", err)
	}
	if _, err := expenses.GetByID(ctx, snapshot.ID); err != nil {
		t.Fatalf("stale delete must not remove the expense: %v", err)
	}

	if err := expenses.DeleteIfUnchanged(ctx, updated); err != nil {
		t.Fatalf("delete updated snapshot: %v", err)
	}
	if _, err := expenses.GetByID(ctx, snapshot.ID); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("expected the expense to be deleted, got %v", err)
	}
	if err := expenses.DeleteIfUnchanged(ctx, updated); !errors.Is(err, model.ErrConflict) {
		t.Fatalf("expected a conflict when deleting twice, got %v", err)
	}
}

func testDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func mustCreateExpense(
	t *testing.T,
	repository *expense.Repository,
	item model.Expense,
) model.Expense {
	t.Helper()

	created, err := repository.Create(context.Background(), item)
	if err != nil {
		t.Fatalf("create expense: %v", err)
	}
	return created
}
