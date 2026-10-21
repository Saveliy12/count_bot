package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	requestIDHeader = "X-Request-ID"
	maxRequestIDLen = 128
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if id == "" || len(id) > maxRequestIDLen {
			id = newRequestID()
		}

		r.Header.Set(requestIDHeader, id)
		w.Header().Set(requestIDHeader, id)
		next.ServeHTTP(w, r)
	})
}

func newRequestID() string {
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(random)
}

func RequestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		writer := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(writer, r)

		status := writer.status
		if status == 0 {
			status = http.StatusOK
		}

		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration", time.Since(startedAt),
			"request_id", r.Header.Get(requestIDHeader),
		)
	})
}

func Recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered",
					"panic", recovered,
					"stack", string(debug.Stack()),
					"request_id", r.Header.Get(requestIDHeader),
				)
				WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func AllowMethods(next http.HandlerFunc, methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if slices.Contains(methods, r.Method) {
			next(w, r)
			return
		}

		w.Header().Set("Allow", strings.Join(methods, ", "))
		WriteError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}
