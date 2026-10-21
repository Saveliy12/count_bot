package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type pingerFunc func(ctx context.Context) error

func (f pingerFunc) PingContext(ctx context.Context) error {
	return f(ctx)
}

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewRouter(
		pingerFunc(func(context.Context) error { return nil }),
		nil,
		logger,
	)
}

func TestNotFound(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/missing", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error.Code != "not_found" {
		t.Fatalf("expected not_found, got %q", body.Error.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/users", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, allow)
	}
}

func TestRouterAddsRequestID(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	testRouter().ServeHTTP(response, request)

	if id := response.Header().Get("X-Request-ID"); id == "" {
		t.Fatal("expected a request ID")
	}
}
