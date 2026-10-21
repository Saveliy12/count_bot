package httpx

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRequestID(t *testing.T) {
	t.Run("generates request ID", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if id := response.Header().Get(requestIDHeader); id == "" {
			t.Fatal("expected a generated request ID")
		}
	})

	t.Run("preserves client request ID", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(requestIDHeader, "client-request-id")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if id := response.Header().Get(requestIDHeader); id != "client-request-id" {
			t.Fatalf("expected client request ID, got %q", id)
		}
	})
}

func TestRecoverer(t *testing.T) {
	handler := Recoverer(discardLogger(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", response.Code)
	}

	var body errorEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Error.Code != "internal_error" {
		t.Fatalf("expected internal_error, got %q", body.Error.Code)
	}
}

func TestAllowMethods(t *testing.T) {
	handler := AllowMethods(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, http.MethodGet)
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", response.Code)
	}
	if allow := response.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, allow)
	}
}
