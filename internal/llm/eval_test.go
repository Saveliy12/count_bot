package llm

import (
	"context"
	"io"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

var evalToday = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

type evalExpectation struct {
	action         Action
	amount         string
	category       model.Category
	dateHint       string
	targetMode     TargetMode
	targetQuery    string
	targetDateHint string
	targetCategory model.Category
	targetID       int64
	periodFrom     string
	periodTo       string
	periodPreset   PeriodPreset
	clearComment   *bool
}

type evalCase struct {
	name string
	text string
	want evalExpectation
}

func TestModelEvaluation(t *testing.T) {
	if os.Getenv("LLM_EVAL") != "1" {
		t.Skip("set LLM_EVAL=1 to run the real model evaluation")
	}

	apiKey := strings.TrimSpace(os.Getenv("GROQ_API_KEY"))
	if apiKey == "" {
		t.Skip("GROQ_API_KEY is not set")
	}

	modelName := strings.TrimSpace(os.Getenv("LLM_MODEL"))
	if modelName == "" {
		modelName = "openai/gpt-oss-120b"
	}
	baseURL := strings.TrimSpace(os.Getenv("LLM_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	client := NewClient(Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   modelName,
		Timeout: 25 * time.Second,
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	cases := []evalCase{
		{name: "coffee", text: "кофе 350", want: evalExpectation{action: ActionCreate, amount: "350", category: model.CategoryFood, dateHint: "today"}},
		{name: "taxi yesterday", text: "такси 1200 вчера", want: evalExpectation{action: ActionCreate, amount: "1200", category: model.CategoryTransport, dateHint: "yesterday"}},
		{name: "groceries", text: "продукты на 2500", want: evalExpectation{action: ActionCreate, amount: "2500", category: model.CategoryFood}},
		{name: "pharmacy", text: "аптека 600", want: evalExpectation{action: ActionCreate, amount: "600", category: model.CategoryHealth}},
		{name: "subscription", text: "подписка яндекс плюс 399", want: evalExpectation{action: ActionCreate, amount: "399", category: model.CategorySubscriptions}},
		{name: "haircut", text: "стрижка 1500", want: evalExpectation{action: ActionCreate, amount: "1500", category: model.CategoryPersonalCare}},
		{name: "movie", text: "кино 800", want: evalExpectation{action: ActionCreate, amount: "800", category: model.CategoryEntertainment}},
		{name: "fuel", text: "бензин 3000", want: evalExpectation{action: ActionCreate, amount: "3000", category: model.CategoryTransport}},
		{name: "book", text: "купил книгу за 1200", want: evalExpectation{action: ActionCreate, amount: "1200", category: model.CategoryEducation}},
		{name: "hotel", text: "отель 15000", want: evalExpectation{action: ActionCreate, amount: "15000", category: model.CategoryTravel}},
		{name: "gift", text: "подарок маме 2000", want: evalExpectation{action: ActionCreate, amount: "2000", category: model.CategoryGifts}},
		{name: "pet food", text: "корм коту 900", want: evalExpectation{action: ActionCreate, amount: "900", category: model.CategoryPets}},
		{name: "gym", text: "спортзал 2500", want: evalExpectation{action: ActionCreate, amount: "2500", category: model.CategorySport}},
		{name: "rent", text: "аренда квартиры 40000", want: evalExpectation{action: ActionCreate, amount: "40000", category: model.CategoryHousing}},
		{name: "unknown category", text: "потратил 500 на что-то", want: evalExpectation{action: ActionCreate, amount: "500", category: model.CategoryOther}},
		{name: "lunch yesterday", text: "вчера обед 800", want: evalExpectation{action: ActionCreate, amount: "800", category: model.CategoryFood, dateHint: "yesterday"}},
		{name: "food month", text: "сколько потратил на еду в этом месяце", want: evalExpectation{action: ActionList, category: model.CategoryFood, periodPreset: PeriodMonth}},
		{name: "week", text: "покажи траты за неделю", want: evalExpectation{action: ActionList, periodPreset: PeriodWeek}},
		{name: "today", text: "траты за сегодня", want: evalExpectation{action: ActionList, periodPreset: PeriodToday}},
		{name: "year", text: "сколько потратил за год", want: evalExpectation{action: ActionList, periodPreset: PeriodYear}},
		{name: "list exact date", text: "покажи траты за 20.10.2026", want: evalExpectation{action: ActionList, periodFrom: "2026-10-20", periodTo: "2026-10-20"}},
		{name: "delete taxi yesterday", text: "удали такси вчера", want: evalExpectation{action: ActionDelete, targetMode: TargetQuery, targetQuery: "такси", targetDateHint: "yesterday"}},
		{name: "delete yesterday food", text: "удали вчерашние траты на еду", want: evalExpectation{action: ActionDelete, targetMode: TargetQuery, targetCategory: model.CategoryFood, targetDateHint: "yesterday"}},
		{name: "update taxi yesterday", text: "измени такси вчера на 1300", want: evalExpectation{action: ActionUpdate, amount: "1300", targetMode: TargetQuery, targetQuery: "такси", targetDateHint: "yesterday"}},
		{name: "update latest yesterday", text: "измени последнюю вчерашнюю трату на 500", want: evalExpectation{action: ActionUpdate, amount: "500", targetMode: TargetLatest, targetDateHint: "yesterday"}},
		{name: "update latest old amount", text: "последнее было не 350, а 450", want: evalExpectation{action: ActionUpdate, amount: "450", targetMode: TargetLatest}},
		{name: "update taxi", text: "измени такси на 1300", want: evalExpectation{action: ActionUpdate, amount: "1300", targetMode: TargetQuery, targetQuery: "такси"}},
		{name: "clear comment", text: "убери комментарий у последнего", want: evalExpectation{action: ActionUpdate, targetMode: TargetLatest, clearComment: boolPtr(true)}},
		{name: "delete latest", text: "удали последнее", want: evalExpectation{action: ActionDelete, targetMode: TargetLatest}},
		{name: "delete taxi", text: "удали такси", want: evalExpectation{action: ActionDelete, targetMode: TargetQuery, targetQuery: "такси"}},
		{name: "delete by id", text: "удали трату 5", want: evalExpectation{action: ActionDelete, targetMode: TargetID, targetID: 5}},
		{name: "delete taxi exact date", text: "удали такси 20.10.2026", want: evalExpectation{action: ActionDelete, targetMode: TargetQuery, targetQuery: "такси", targetDateHint: "2026-10-20"}},
		{name: "delete bare id", text: "удали 6", want: evalExpectation{action: ActionDelete, targetMode: TargetID, targetID: 6}},
		{name: "clarify empty expense", text: "запиши трату", want: evalExpectation{action: ActionClarify}},
		{name: "clarify no amount", text: "потратил", want: evalExpectation{action: ActionClarify}},
		{name: "clarify no amount with comment", text: "кофе", want: evalExpectation{action: ActionClarify}},
	}

	if rawOffset := strings.TrimSpace(os.Getenv("LLM_EVAL_OFFSET")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 || offset >= len(cases) {
			t.Fatalf("invalid LLM_EVAL_OFFSET %q", rawOffset)
		}
		cases = cases[offset:]
	}

	if rawLimit := strings.TrimSpace(os.Getenv("LLM_EVAL_LIMIT")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 || limit > len(cases) {
			t.Fatalf("invalid LLM_EVAL_LIMIT %q", rawLimit)
		}
		cases = cases[:limit]
	}

	delay := 2 * time.Second
	if rawDelay := strings.TrimSpace(os.Getenv("LLM_EVAL_DELAY")); rawDelay != "" {
		parsed, err := time.ParseDuration(rawDelay)
		if err != nil || parsed < 0 {
			t.Fatalf("invalid LLM_EVAL_DELAY %q", rawDelay)
		}
		delay = parsed
	}

	ctx := context.Background()
	passed := 0
	var durations []time.Duration
	for _, test := range cases {
		startedAt := time.Now()
		intent, err := client.Parse(ctx, Request{Text: test.text, Today: evalToday})
		elapsed := time.Since(startedAt)
		durations = append(durations, elapsed)
		if err != nil {
			t.Errorf("%s: parse %q: %v", test.name, test.text, err)
			continue
		}

		ok := true
		if intent.Action != test.want.action {
			t.Errorf("%s: action: got %q, want %q", test.name, intent.Action, test.want.action)
			ok = false
		}
		if test.want.amount != "" {
			want := decimal.RequireFromString(test.want.amount)
			if intent.Amount == nil || !intent.Amount.Equal(want) {
				t.Errorf("%s: amount: got %v, want %s", test.name, intent.Amount, want)
				ok = false
			}
		}
		if test.want.category != "" {
			if intent.Category == nil || *intent.Category != test.want.category {
				t.Errorf("%s: category: got %v, want %q", test.name, intent.Category, test.want.category)
				ok = false
			}
		}
		if test.want.dateHint != "" {
			if intent.DateHint == nil || *intent.DateHint != test.want.dateHint {
				t.Errorf("%s: date hint: got %v, want %q", test.name, intent.DateHint, test.want.dateHint)
				ok = false
			}
		}
		if test.want.targetMode != "" && intent.TargetMode != test.want.targetMode {
			t.Errorf("%s: target mode: got %q, want %q", test.name, intent.TargetMode, test.want.targetMode)
			ok = false
		}
		if test.want.targetQuery != "" {
			if intent.TargetQuery == nil || !strings.Contains(strings.ToLower(*intent.TargetQuery), test.want.targetQuery) {
				t.Errorf("%s: target query: got %v, want %q", test.name, intent.TargetQuery, test.want.targetQuery)
				ok = false
			}
		}
		if test.want.targetDateHint != "" {
			if intent.TargetDateHint == nil || *intent.TargetDateHint != test.want.targetDateHint {
				t.Errorf("%s: target date hint: got %v, want %q", test.name, intent.TargetDateHint, test.want.targetDateHint)
				ok = false
			}
		}
		if test.want.targetCategory != "" {
			if intent.TargetCategory == nil || *intent.TargetCategory != test.want.targetCategory {
				t.Errorf("%s: target category: got %v, want %q", test.name, intent.TargetCategory, test.want.targetCategory)
				ok = false
			}
		}
		if test.want.targetID != 0 {
			if intent.TargetID == nil || *intent.TargetID != test.want.targetID {
				t.Errorf("%s: target id: got %v, want %d", test.name, intent.TargetID, test.want.targetID)
				ok = false
			}
		}
		if test.want.periodFrom != "" {
			if intent.PeriodFrom == nil || intent.PeriodFrom.Format(dateLayout) != test.want.periodFrom {
				t.Errorf("%s: period from: got %v, want %q", test.name, intent.PeriodFrom, test.want.periodFrom)
				ok = false
			}
		}
		if test.want.periodTo != "" {
			if intent.PeriodTo == nil || intent.PeriodTo.Format(dateLayout) != test.want.periodTo {
				t.Errorf("%s: period to: got %v, want %q", test.name, intent.PeriodTo, test.want.periodTo)
				ok = false
			}
		}
		if test.want.periodPreset != "" && intent.PeriodPreset != test.want.periodPreset {
			t.Errorf("%s: period: got %q, want %q", test.name, intent.PeriodPreset, test.want.periodPreset)
			ok = false
		}
		if test.want.clearComment != nil && intent.ClearComment != *test.want.clearComment {
			t.Errorf("%s: clear_comment: got %t, want %t", test.name, intent.ClearComment, *test.want.clearComment)
			ok = false
		}
		if ok {
			passed++
		}

		t.Logf("%s: %.2fs", test.name, elapsed.Seconds())
		time.Sleep(delay)
	}

	if len(durations) > 0 {
		sorted := slices.Clone(durations)
		slices.Sort(sorted)
		median := sorted[len(sorted)/2]
		t.Logf("latency: first=%.2fs median=%.2fs max=%.2fs",
			durations[0].Seconds(), median.Seconds(), sorted[len(sorted)-1].Seconds())
	}

	t.Logf("accuracy: %d/%d", passed, len(cases))
	if passed != len(cases) {
		t.Fatalf("model evaluation failed: %d/%d cases passed", passed, len(cases))
	}
}

func boolPtr(value bool) *bool {
	return &value
}
