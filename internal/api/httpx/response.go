package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/Saveliy12/expenses/internal/model"
)

const maxRequestBodySize = 1 << 20

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode JSON response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, errorEnvelope{
		Error: errorBody{
			Code:    code,
			Message: message,
		},
	})
}

func WriteValidationError(w http.ResponseWriter, field, message string) {
	WriteJSON(w, http.StatusBadRequest, errorEnvelope{
		Error: errorBody{
			Code:    "validation_error",
			Message: message,
			Field:   field,
		},
	})
}

func WriteServiceError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var validationErr *model.ValidationError
	if errors.As(err, &validationErr) {
		WriteValidationError(w, validationErr.Field, validationErr.Message)
		return
	}
	if errors.Is(err, model.ErrValidation) {
		WriteError(w, http.StatusBadRequest, "validation_error", "validation error")
		return
	}
	if errors.Is(err, model.ErrNotFound) {
		WriteError(w, http.StatusNotFound, "not_found", "resource not found")
		return
	}
	if errors.Is(err, model.ErrConflict) {
		WriteError(w, http.StatusConflict, "conflict", "resource has changed")
		return
	}

	logger.Error("request failed", "error", err)
	WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}

	return nil
}
