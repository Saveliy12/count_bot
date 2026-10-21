package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientRetriesGenerationFailure(t *testing.T) {
	content := `{
		"action":"list",
		"amount":null,
		"old_amount":null,
		"category":"food",
		"date_hint":null,
		"comment":null,
		"clear_comment":false,
		"target_mode":null,
		"target_id":null,
		"target_query":null,
		"target_date_hint":null,
		"target_category":null,
		"period_preset":"month",
		"period_from":null,
		"period_to":null,
		"clarification":null
	}`

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":"json_validate_failed","failed_generation":"{"}}`))
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

	intent, err := client.Parse(context.Background(), Request{
		Text:  "сколько потратил на еду",
		Today: time.Now(),
	})
	if err != nil {
		t.Fatalf("expected a successful retry, got %v", err)
	}
	if intent.Action != ActionList {
		t.Fatalf("unexpected action %q", intent.Action)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}
