package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/llm"
	"github.com/Saveliy12/expenses/internal/model"
	"github.com/Saveliy12/expenses/internal/service"
)

const (
	isoDateLayout         = "2006-01-02"
	displayDateLayout     = "02.01.2006"
	listItemLimit         = 10
	displayedCommentLimit = 80
	targetSearchPageSize  = model.MaxPageSize
	targetSearchMaxPages  = 5
	targetCandidateLimit  = 5
)

var (
	errTargetMissing     = errors.New("target is missing")
	errTargetNotFound    = errors.New("target not found")
	errTargetAmbiguous   = errors.New("target is ambiguous")
	errTargetTooBroad    = errors.New("target search is too broad")
	errTargetInvalidDate = errors.New("target date is invalid")
)

type ambiguousTargetError struct {
	matches []model.Expense
}

func (e *ambiguousTargetError) Error() string {
	return "target is ambiguous"
}

func (e *ambiguousTargetError) Unwrap() error {
	return errTargetAmbiguous
}

type LLMProcessor struct {
	provider llm.Provider
	expenses service.ExpenseService
	pending  *pendingStates
	logger   *slog.Logger
	now      func() time.Time
}

func NewLLMProcessor(
	provider llm.Provider,
	expenses service.ExpenseService,
	location *time.Location,
	logger *slog.Logger,
) *LLMProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	if location == nil {
		location = time.Local
	}

	return &LLMProcessor{
		provider: provider,
		expenses: expenses,
		pending:  newPendingStates(),
		logger:   logger,
		now: func() time.Time {
			return time.Now().In(location)
		},
	}
}

func (p *LLMProcessor) ResetPending(telegramID int64) {
	p.pending.clear(telegramID)
}

func (p *LLMProcessor) Process(ctx context.Context, actor model.Actor, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}

	now := p.now()
	if state, ok := p.pending.take(actor.TelegramID, now); ok {
		switch state.kind {
		case pendingKindDeletion:
			switch {
			case isYes(text):
				return p.deleteConfirmed(ctx, actor, state.expense)
			case isNo(text):
				return "Отменил удаление.", nil
			}
		case pendingKindClarification:
			text = clarificationContext(state, text)
		case pendingKindTargetSelection:
			reply, handled, err := p.selectTarget(ctx, actor, state, text, now)
			if err != nil {
				return "", err
			}
			if handled {
				return reply, nil
			}
		}
	}

	intent, err := p.provider.Parse(ctx, llm.Request{Text: text, Today: now})
	if err != nil {
		switch {
		case errors.Is(err, llm.ErrRateLimited):
			p.logger.Warn("llm rate limited", "error", err)
			return "Слишком много запросов к ИИ. Попробуй через минуту.", nil
		case errors.Is(err, llm.ErrUnavailable):
			p.logger.Warn("llm unavailable", "error", err)
			return "ИИ временно недоступен. Попробуй ещё раз.", nil
		case errors.Is(err, llm.ErrInvalidResponse):
			p.logger.Error("invalid llm response", "error", err)
			return "Не понял сообщение. Попробуй переформулировать.", nil
		default:
			return "", fmt.Errorf("parse intent: %w", err)
		}
	}

	if intent.Action == llm.ActionClarify {
		question := p.clarify(intent)
		p.pending.putClarification(actor.TelegramID, text, question, now)
		return question, nil
	}

	switch intent.Action {
	case llm.ActionCreate:
		return p.create(ctx, actor, intent, now)
	case llm.ActionList:
		return p.list(ctx, actor, intent, now)
	case llm.ActionUpdate:
		return p.update(ctx, actor, intent, now)
	case llm.ActionDelete:
		return p.delete(ctx, actor, intent, now)
	default:
		return "Не понял, что сделать. Напиши, например: «кофе 350».", nil
	}
}

func clarificationContext(state pendingState, answer string) string {
	return fmt.Sprintf(
		"Предыдущее сообщение пользователя: %s\nВопрос бота: %s\nОтвет пользователя: %s",
		state.originalText,
		state.question,
		answer,
	)
}

func (p *LLMProcessor) create(
	ctx context.Context,
	actor model.Actor,
	intent llm.Intent,
	now time.Time,
) (string, error) {
	if intent.Amount == nil {
		return "Не понял сумму. Напиши, например: «кофе 350».", nil
	}

	if intent.Category == nil {
		return "К какой категории отнести трату? Напиши, например: «еда», «транспорт» или «другое».", nil
	}
	category := *intent.Category

	spentAt, err := expenseDate(intent.DateHint, now)
	if err != nil {
		return "Не понял дату. Напиши её как 2026-10-21.", nil
	}

	comment := ""
	if intent.Comment != nil {
		comment = *intent.Comment
	}

	expense, err := p.expenses.CreateExpense(ctx, actor, model.CreateExpenseInput{
		Amount:   *intent.Amount,
		Category: category,
		SpentAt:  spentAt,
		Comment:  comment,
	})
	if err != nil {
		return p.fail("create", err)
	}

	return "Записал: " + expenseDescription(expense) + ".", nil
}

func (p *LLMProcessor) list(
	ctx context.Context,
	actor model.Actor,
	intent llm.Intent,
	now time.Time,
) (string, error) {
	from, to, label, err := periodRange(intent, now)
	if err != nil {
		return "Не понял период. Скажи, например: «за этот месяц».", nil
	}

	filter := model.ExpenseFilter{
		From:     from,
		To:       to,
		Category: intent.Category,
	}
	summary, err := p.expenses.SummarizeExpenses(ctx, actor, filter)
	if err != nil {
		return "", fmt.Errorf("summarize expenses: %w", err)
	}
	if summary.Count == 0 {
		return fmt.Sprintf("За %s трат нет.", label), nil
	}

	filter.Limit = listItemLimit
	page, err := p.expenses.ListExpenses(ctx, actor, filter)
	if err != nil {
		return "", fmt.Errorf("list expenses: %w", err)
	}

	var builder strings.Builder
	fmt.Fprintf(
		&builder,
		"За %s: %s, трат: %d.",
		label,
		formatMoney(summary.Total),
		summary.Count,
	)
	for _, expense := range page.Items {
		fmt.Fprintf(
			&builder,
			"\n• #%d %s — %s, %s%s",
			expense.ID,
			expense.SpentAt.Format("02.01"),
			formatMoney(expense.Amount),
			expense.Category.Label(),
			commentSuffix(expense.Comment),
		)
	}
	if remaining := summary.Count - int64(len(page.Items)); remaining > 0 {
		fmt.Fprintf(&builder, "\n…и ещё %d.", remaining)
	}

	return builder.String(), nil
}

func (p *LLMProcessor) update(
	ctx context.Context,
	actor model.Actor,
	intent llm.Intent,
	now time.Time,
) (string, error) {
	patch := model.UpdateExpenseInput{
		Amount:   intent.Amount,
		Category: intent.Category,
		Comment:  intent.Comment,
	}
	if intent.ClearComment {
		empty := ""
		patch.Comment = &empty
	}
	if intent.DateHint != nil {
		spentAt, err := expenseDate(intent.DateHint, now)
		if err != nil {
			return "Не понял дату. Напиши её как 2026-10-21.", nil
		}
		patch.SpentAt = &spentAt
	}
	if patch.Amount == nil && patch.Category == nil && patch.Comment == nil && patch.SpentAt == nil {
		return "Что именно изменить: сумму, категорию, дату или комментарий?", nil
	}

	target, err := p.resolveTarget(ctx, actor, intent, now)
	if err != nil {
		if reply, ok := p.handleAmbiguousTarget(actor, err, intent, patch, now); ok {
			return reply, nil
		}
		return p.targetFailure(err)
	}
	if intent.OldAmount != nil && !target.Amount.Equal(*intent.OldAmount) {
		return fmt.Sprintf(
			"У траты #%d сумма %s, а не %s. Уточни, какую трату изменить.",
			target.ID,
			formatMoney(target.Amount),
			formatMoney(*intent.OldAmount),
		), nil
	}

	updated, err := p.expenses.UpdateExpense(ctx, actor, target.ID, patch)
	if err != nil {
		return p.fail("update", err)
	}

	return "Изменил: " + expenseDescription(updated) + ".", nil
}

func (p *LLMProcessor) delete(
	ctx context.Context,
	actor model.Actor,
	intent llm.Intent,
	now time.Time,
) (string, error) {
	target, err := p.resolveTarget(ctx, actor, intent, now)
	if err != nil {
		if reply, ok := p.handleAmbiguousTarget(actor, err, intent, model.UpdateExpenseInput{}, now); ok {
			return reply, nil
		}
		return p.targetFailure(err)
	}

	p.pending.putDeletion(actor.TelegramID, target, now)

	return fmt.Sprintf(
		"Удалить: %s? Напиши «да» или «нет».",
		expenseDescription(target),
	), nil
}

func (p *LLMProcessor) deleteConfirmed(
	ctx context.Context,
	actor model.Actor,
	expense model.Expense,
) (string, error) {
	if err := p.expenses.DeleteExpenseIfUnchanged(ctx, actor, expense); err != nil {
		return p.fail("delete", err)
	}

	return "Удалил: " + expenseDescription(expense) + ".", nil
}

func (p *LLMProcessor) clarify(intent llm.Intent) string {
	if intent.Clarification != "" {
		return intent.Clarification
	}

	return "Уточни, пожалуйста: что за трата и на какую сумму?"
}

func (p *LLMProcessor) resolveTarget(
	ctx context.Context,
	actor model.Actor,
	intent llm.Intent,
	now time.Time,
) (model.Expense, error) {
	filter, err := targetFilter(intent, now)
	if err != nil {
		return model.Expense{}, err
	}

	switch intent.TargetMode {
	case llm.TargetLatest:
		if filter.Category == nil && filter.From.IsZero() && filter.To.IsZero() {
			expense, err := p.expenses.LatestExpense(ctx, actor)
			if errors.Is(err, model.ErrNotFound) {
				return model.Expense{}, errTargetNotFound
			}
			if err != nil {
				return model.Expense{}, err
			}

			return expense, nil
		}

		expense, err := p.expenses.LatestExpenseFiltered(ctx, actor, filter)
		if errors.Is(err, model.ErrNotFound) {
			return model.Expense{}, errTargetNotFound
		}
		if err != nil {
			return model.Expense{}, err
		}

		return expense, nil
	case llm.TargetID:
		if intent.TargetID == nil {
			return model.Expense{}, errTargetMissing
		}
		expense, err := p.expenses.GetExpense(ctx, actor, *intent.TargetID)
		if errors.Is(err, model.ErrNotFound) {
			return model.Expense{}, errTargetNotFound
		}
		if err != nil {
			return model.Expense{}, err
		}

		return expense, nil
	case llm.TargetQuery:
		matches, err := p.findTargetMatches(ctx, actor, intent.TargetQuery, intent.OldAmount, filter)
		if err != nil {
			return model.Expense{}, err
		}

		switch len(matches) {
		case 0:
			return model.Expense{}, errTargetNotFound
		case 1:
			return matches[0], nil
		default:
			return model.Expense{}, &ambiguousTargetError{matches: matches}
		}
	default:
		return model.Expense{}, errTargetMissing
	}
}

func (p *LLMProcessor) findTargetMatches(
	ctx context.Context,
	actor model.Actor,
	query *string,
	oldAmount *decimal.Decimal,
	filter model.ExpenseFilter,
) ([]model.Expense, error) {
	var normalizedQuery string
	if query != nil {
		normalizedQuery = strings.ToLower(strings.TrimSpace(*query))
	}

	var matches []model.Expense
	offset := 0
	exhausted := false

	for pageIndex := 0; pageIndex < targetSearchMaxPages; pageIndex++ {
		filter.Limit = targetSearchPageSize
		filter.Offset = offset

		page, err := p.expenses.ListExpenses(ctx, actor, filter)
		if err != nil {
			return nil, fmt.Errorf("find expense target: %w", err)
		}

		for _, expense := range page.Items {
			if normalizedQuery != "" && !expenseMatches(expense, normalizedQuery) {
				continue
			}
			if oldAmount != nil && !expense.Amount.Equal(*oldAmount) {
				continue
			}
			matches = append(matches, expense)
			if len(matches) >= targetCandidateLimit {
				return matches, nil
			}
		}

		if !page.HasMore {
			exhausted = true
			break
		}
		if len(page.Items) == 0 {
			break
		}
		offset += len(page.Items)
	}

	if !exhausted {
		return matches, errTargetTooBroad
	}

	return matches, nil
}

func targetFilter(intent llm.Intent, now time.Time) (model.ExpenseFilter, error) {
	var filter model.ExpenseFilter

	if intent.TargetDateHint != nil {
		date, err := expenseDate(intent.TargetDateHint, now)
		if err != nil {
			return model.ExpenseFilter{}, fmt.Errorf("%w: %v", errTargetInvalidDate, err)
		}
		filter.From = date
		filter.To = date
	}
	filter.Category = intent.TargetCategory

	return filter, nil
}

func (p *LLMProcessor) handleAmbiguousTarget(
	actor model.Actor,
	err error,
	intent llm.Intent,
	patch model.UpdateExpenseInput,
	now time.Time,
) (string, bool) {
	var ambiguous *ambiguousTargetError
	if !errors.As(err, &ambiguous) {
		return "", false
	}

	p.pending.putTargetSelection(actor.TelegramID, ambiguous.matches, intent.Action, patch, now)

	return p.ambiguousMessage(ambiguous.matches, intent), true
}

func (p *LLMProcessor) ambiguousMessage(matches []model.Expense, intent llm.Intent) string {
	var builder strings.Builder
	builder.WriteString("Нашёл несколько трат:")
	for _, expense := range matches {
		fmt.Fprintf(
			&builder,
			"\n• #%d %s — %s, %s%s",
			expense.ID,
			formatDisplayDate(expense.SpentAt),
			formatMoney(expense.Amount),
			expense.Category.Label(),
			commentSuffix(expense.Comment),
		)
	}

	builder.WriteString("\nНапиши номер из списка: 6.")

	if intent.Action == llm.ActionUpdate {
		example := "измени такси 20.10.2026"
		if intent.Amount != nil {
			example += " на " + formatMoney(*intent.Amount)
		}
		builder.WriteString(" Или уточни дату: " + example + ".")
	} else {
		builder.WriteString(" Или уточни дату: удали такси 20.10.2026.")
	}

	return builder.String()
}

func (p *LLMProcessor) selectTarget(
	ctx context.Context,
	actor model.Actor,
	state pendingState,
	text string,
	now time.Time,
) (string, bool, error) {
	if isNo(text) {
		return "Отменил выбор.", true, nil
	}
	if isYes(text) {
		return "Сначала выбери номер траты из списка.", true, nil
	}

	id, ok := targetSelectionID(text)
	if !ok {
		return "", false, nil
	}

	var selected *model.Expense
	for index := range state.candidates {
		if state.candidates[index].ID == id {
			selected = &state.candidates[index]
			break
		}
	}
	if selected == nil {
		p.pending.putTargetSelection(actor.TelegramID, state.candidates, state.action, state.patch, now)
		return fmt.Sprintf(
			"В списке нет траты #%d. Напиши номер из списка или «отмена».",
			id,
		), true, nil
	}

	switch state.action {
	case llm.ActionDelete:
		p.pending.putDeletion(actor.TelegramID, *selected, now)
		return fmt.Sprintf(
			"Удалить: %s? Напиши «да» или «нет».",
			expenseDescription(*selected),
		), true, nil
	case llm.ActionUpdate:
		updated, err := p.expenses.UpdateExpense(ctx, actor, selected.ID, state.patch)
		if err != nil {
			reply, failure := p.fail("update", err)
			return reply, true, failure
		}
		return "Изменил: " + expenseDescription(updated) + ".", true, nil
	default:
		return "", false, nil
	}
}

func targetSelectionID(text string) (int64, bool) {
	value := strings.ToLower(strings.TrimSpace(text))
	value = strings.TrimPrefix(value, "номер")
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "#")
	value = strings.TrimSpace(value)

	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

func (p *LLMProcessor) targetFailure(err error) (string, error) {
	switch {
	case errors.Is(err, errTargetMissing):
		return "Не понял, какую трату имеешь в виду. Скажи «последнее» или назови описание.", nil
	case errors.Is(err, errTargetNotFound):
		return "Не нашёл такую трату. Уточни описание или скажи «последнее».", nil
	case errors.Is(err, errTargetAmbiguous):
		return "Нашёл несколько подходящих трат. Скажи «последнее» или назови номер траты.", nil
	case errors.Is(err, errTargetTooBroad):
		return "Слишком много подходящих трат. Уточни описание или назови номер.", nil
	case errors.Is(err, errTargetInvalidDate):
		return "Не понял дату старой траты. Напиши её как 2026-10-21.", nil
	default:
		return "", err
	}
}

func (p *LLMProcessor) fail(action string, err error) (string, error) {
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) {
		return validationMessage(validationErr), nil
	}
	if errors.Is(err, model.ErrNotFound) {
		return "Не нашёл такую трату.", nil
	}
	if errors.Is(err, model.ErrConflict) {
		return "Трата изменилась или уже удалена. Запроси удаление заново.", nil
	}

	p.logger.Error("expense action failed", "action", action, "error", err)
	return "", fmt.Errorf("%s expense: %w", action, err)
}

func periodRange(intent llm.Intent, now time.Time) (time.Time, time.Time, string, error) {
	today := expenseDateOnly(now)

	if intent.PeriodFrom != nil || intent.PeriodTo != nil {
		var from, to time.Time
		if intent.PeriodFrom != nil {
			from = *intent.PeriodFrom
		}
		if intent.PeriodTo != nil {
			to = *intent.PeriodTo
		}

		switch {
		case from.IsZero() && to.IsZero():
			return firstOfMonth(today), today, "этот месяц", nil
		case from.IsZero():
			return time.Time{}, to, "до " + to.Format(displayDateLayout), nil
		case to.IsZero():
			return from, today, "с " + from.Format(displayDateLayout), nil
		default:
			if from.Equal(to) {
				return from, to, from.Format(displayDateLayout), nil
			}
			return from, to, from.Format(displayDateLayout) + " — " + to.Format(displayDateLayout), nil
		}
	}

	switch intent.PeriodPreset {
	case llm.PeriodToday:
		return today, today, "сегодня", nil
	case llm.PeriodYesterday:
		yesterday := today.AddDate(0, 0, -1)
		return yesterday, yesterday, "вчера", nil
	case llm.PeriodWeek:
		return today.AddDate(0, 0, -6), today, "последние 7 дней", nil
	case llm.PeriodMonth, "":
		return firstOfMonth(today), today, "этот месяц", nil
	case llm.PeriodYear:
		return time.Date(today.Year(), time.January, 1, 0, 0, 0, 0, time.UTC), today, "этот год", nil
	case llm.PeriodAll:
		return time.Time{}, time.Time{}, "всё время", nil
	default:
		return time.Time{}, time.Time{}, "", fmt.Errorf("unsupported period %q", intent.PeriodPreset)
	}
}

func expenseDate(hint *string, now time.Time) (time.Time, error) {
	today := expenseDateOnly(now)
	if hint == nil {
		return today, nil
	}

	switch strings.ToLower(strings.TrimSpace(*hint)) {
	case "", "today":
		return today, nil
	case "yesterday":
		return today.AddDate(0, 0, -1), nil
	default:
		return time.Parse(isoDateLayout, strings.TrimSpace(*hint))
	}
}

func expenseDateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func firstOfMonth(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func expenseMatches(expense model.Expense, query string) bool {
	if query == "" {
		return false
	}

	if strings.Contains(strings.ToLower(expense.Comment), query) ||
		strings.Contains(strings.ToLower(expense.Category.Label()), query) ||
		strings.Contains(strings.ToLower(string(expense.Category)), query) {
		return true
	}

	amount, err := parseQueryAmount(query)
	if err != nil {
		return false
	}

	return amount.Equal(expense.Amount)
}

func parseQueryAmount(query string) (decimal.Decimal, error) {
	value := strings.ReplaceAll(query, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")
	value = strings.TrimSuffix(value, "₽")
	if strings.Count(value, ",") == 1 && !strings.Contains(value, ".") {
		value = strings.Replace(value, ",", ".", 1)
	}

	return decimal.NewFromString(value)
}

func expenseDescription(expense model.Expense) string {
	return fmt.Sprintf(
		"%s — %s, %s%s",
		formatMoney(expense.Amount),
		expense.Category.Label(),
		formatDisplayDate(expense.SpentAt),
		commentSuffix(expense.Comment),
	)
}

func isYes(text string) bool {
	switch normalizeAnswer(text) {
	case "да", "yes", "подтверждаю", "подтвердить", "удаляй", "ок", "ok":
		return true
	default:
		return false
	}
}

func isNo(text string) bool {
	switch normalizeAnswer(text) {
	case "нет", "no", "отмена", "отменить", "не надо":
		return true
	default:
		return false
	}
}

func normalizeAnswer(text string) string {
	return strings.Trim(strings.ToLower(strings.TrimSpace(text)), " .!?,")
}

func formatMoney(value decimal.Decimal) string {
	if value.Equal(value.Round(0)) {
		return value.StringFixed(0) + " ₽"
	}

	return value.StringFixed(2) + " ₽"
}

func formatDisplayDate(value time.Time) string {
	return value.Format(displayDateLayout)
}

func commentSuffix(comment string) string {
	comment = strings.TrimSpace(comment)
	if comment == "" {
		return ""
	}

	return " (" + truncateRunes(comment, displayedCommentLimit) + ")"
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return string(runes[:limit-1]) + "…"
}

func validationMessage(err *model.ValidationError) string {
	switch err.Field {
	case "amount":
		return "Сумма должна быть положительной, не больше 9 999 999 999.99 и не больше двух знаков после запятой."
	case "category":
		return "Неизвестная категория."
	case "spent_at":
		return "Некорректная дата."
	case "comment":
		return "Комментарий слишком длинный."
	case "patch":
		return "Не указано, что именно изменить."
	default:
		return "Проверь данные и попробуй ещё раз."
	}
}
