package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Saveliy12/expenses/internal/model"
)

type capturedRequest struct {
	Authorization string
	UserAgent     string
	Payload       requestPayload
}

type requestPayload struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	Messages    []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	ResponseFormat struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Name   string         `json:"name"`
			Strict bool           `json:"strict"`
			Schema map[string]any `json:"schema"`
		} `json:"json_schema"`
	} `json:"response_format"`
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestClient(t *testing.T, status int, content string) (*Client, <-chan capturedRequest) {
	t.Helper()

	received := make(chan capturedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload requestPayload
		_ = json.NewDecoder(r.Body).Decode(&payload)
		received <- capturedRequest{
			Authorization: r.Header.Get("Authorization"),
			UserAgent:     r.Header.Get("User-Agent"),
			Payload:       payload,
		}

		if status != http.StatusOK {
			w.Header().Set("Retry-After", "10")
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"message":"request failed"}}`))
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{
				map[string]any{
					"message": map[string]any{"content": content},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	return NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 2 * time.Second,
	}, testLogger()), received
}

func TestClientParseCreate(t *testing.T) {
	content := `{
		"action":"create",
		"amount":"350",
		"category":"food",
		"date_hint":"today",
		"comment":"кофе",
		"target_mode":null,
		"target_id":null,
		"target_query":null,
		"period_preset":null,
		"period_from":null,
		"period_to":null,
		"clarification":null
	}`
	client, received := newTestClient(t, http.StatusOK, content)

	intent, err := client.Parse(context.Background(), Request{
		Text:  "кофе 350",
		Today: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("parse intent: %v", err)
	}
	if intent.Action != ActionCreate {
		t.Fatalf("unexpected action %q", intent.Action)
	}
	if intent.Amount == nil || !intent.Amount.Equal(decimal.NewFromInt(350)) {
		t.Fatalf("unexpected amount %v", intent.Amount)
	}
	if intent.Category == nil || *intent.Category != "food" {
		t.Fatalf("unexpected category %v", intent.Category)
	}
	if intent.DateHint == nil || *intent.DateHint != "today" {
		t.Fatalf("unexpected date hint %v", intent.DateHint)
	}
	if intent.Comment == nil || *intent.Comment != "кофе" {
		t.Fatalf("unexpected comment %v", intent.Comment)
	}

	captured := <-received
	if captured.Authorization != "Bearer test-key" {
		t.Fatalf("unexpected authorization header %q", captured.Authorization)
	}
	if captured.UserAgent != "FamilyExpensesBot/1.0" {
		t.Fatalf("unexpected user agent %q", captured.UserAgent)
	}
	if captured.Payload.Model != "test-model" {
		t.Fatalf("unexpected model %q", captured.Payload.Model)
	}
	if captured.Payload.ResponseFormat.Type != "json_schema" {
		t.Fatalf("unexpected response format %q", captured.Payload.ResponseFormat.Type)
	}
	if !captured.Payload.ResponseFormat.JSONSchema.Strict {
		t.Fatal("expected strict schema")
	}
	if !strings.Contains(captured.Payload.Messages[0].Content, "2026-10-21") {
		t.Fatal("system prompt must contain today's date")
	}
	if captured.Payload.Messages[1].Content != "кофе 350" {
		t.Fatalf("unexpected user message %q", captured.Payload.Messages[1].Content)
	}
}

func TestClientParseAmountAndPeriod(t *testing.T) {
	content := `{
		"action":"list",
		"amount":"1 200,50",
		"old_amount":"1 000",
		"category":"food",
		"date_hint":null,
		"comment":null,
		"clear_comment":true,
		"target_mode":null,
		"target_id":null,
		"target_query":null,
		"period_preset":null,
		"period_from":"2026-10-01",
		"period_to":"2026-10-21",
		"clarification":null
	}`
	client, _ := newTestClient(t, http.StatusOK, content)

	intent, err := client.Parse(context.Background(), Request{Text: "еда", Today: time.Now()})
	if err != nil {
		t.Fatalf("parse intent: %v", err)
	}
	if !intent.ClearComment {
		t.Fatal("expected clear_comment to be parsed")
	}
	if intent.Amount == nil || !intent.Amount.Equal(decimal.RequireFromString("1200.50")) {
		t.Fatalf("unexpected amount %v", intent.Amount)
	}
	if intent.OldAmount == nil || !intent.OldAmount.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("unexpected old amount %v", intent.OldAmount)
	}
	if intent.PeriodFrom == nil || intent.PeriodFrom.Format(dateLayout) != "2026-10-01" {
		t.Fatalf("unexpected period from %v", intent.PeriodFrom)
	}
	if intent.PeriodTo == nil || intent.PeriodTo.Format(dateLayout) != "2026-10-21" {
		t.Fatalf("unexpected period to %v", intent.PeriodTo)
	}
}

func TestClientRateLimited(t *testing.T) {
	client, _ := newTestClient(t, http.StatusTooManyRequests, "")

	_, err := client.Parse(context.Background(), Request{Text: "кофе 350", Today: time.Now()})
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestClientRetriesRateLimit(t *testing.T) {
	content := `{
		"action":"clarify",
		"amount":null,
		"category":null,
		"date_hint":null,
		"comment":null,
		"target_mode":null,
		"target_id":null,
		"target_query":null,
		"period_preset":null,
		"period_from":null,
		"period_to":null,
		"clarification":"Сколько?"
	}`

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{
				map[string]any{"message": map[string]any{"content": content}},
			},
		})
	}))
	t.Cleanup(server.Close)

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 2 * time.Second,
	}, testLogger())

	intent, err := client.Parse(context.Background(), Request{Text: "запиши", Today: time.Now()})
	if err != nil {
		t.Fatalf("expected a successful retry, got %v", err)
	}
	if intent.Action != ActionClarify {
		t.Fatalf("unexpected action %q", intent.Action)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func TestClientTotalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	client := NewClient(Config{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "test-model",
		Timeout: 50 * time.Millisecond,
	}, testLogger())

	started := time.Now()
	_, err := client.Parse(context.Background(), Request{Text: "кофе 350", Today: time.Now()})
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("parse exceeded the total timeout: %s", elapsed)
	}
}

func TestClientInvalidContent(t *testing.T) {
	client, _ := newTestClient(t, http.StatusOK, "not a json object")

	_, err := client.Parse(context.Background(), Request{Text: "кофе 350", Today: time.Now()})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}

func TestClientParseTargetFilters(t *testing.T) {
	content := `{
		"action":"delete",
		"amount":null,
		"old_amount":null,
		"category":null,
		"date_hint":null,
		"comment":null,
		"clear_comment":false,
		"target_mode":"query",
		"target_id":null,
		"target_query":null,
		"target_date_hint":"yesterday",
		"target_category":"food",
		"period_preset":null,
		"period_from":null,
		"period_to":null,
		"clarification":null
	}`
	client, _ := newTestClient(t, http.StatusOK, content)

	intent, err := client.Parse(context.Background(), Request{Text: "удали вчерашние траты на еду", Today: time.Now()})
	if err != nil {
		t.Fatalf("parse intent: %v", err)
	}
	if intent.TargetMode != TargetQuery {
		t.Fatalf("unexpected target mode %q", intent.TargetMode)
	}
	if intent.TargetDateHint == nil || *intent.TargetDateHint != "yesterday" {
		t.Fatalf("unexpected target date hint %v", intent.TargetDateHint)
	}
	if intent.TargetCategory == nil || *intent.TargetCategory != model.CategoryFood {
		t.Fatalf("unexpected target category %v", intent.TargetCategory)
	}
}

func TestClientInvalidTarget(t *testing.T) {
	content := `{
		"action":"delete",
		"amount":null,
		"category":null,
		"date_hint":null,
		"comment":null,
		"target_mode":"id",
		"target_id":null,
		"target_query":null,
		"period_preset":null,
		"period_from":null,
		"period_to":null,
		"clarification":null
	}`
	client, _ := newTestClient(t, http.StatusOK, content)

	_, err := client.Parse(context.Background(), Request{Text: "удали", Today: time.Now()})
	if !errors.Is(err, ErrInvalidResponse) {
		t.Fatalf("expected ErrInvalidResponse, got %v", err)
	}
}
