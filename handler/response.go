package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/riskpoint-per/hotel-booking/model"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// ErrorResponse is the JSON body returned for every error
type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			slog.Error("encode response", "error", err)
		}
	}
}

func writeError(w http.ResponseWriter, status int, code, msg string, fields map[string]string) {
	writeJSON(w, status, ErrorResponse{Error: code, Message: msg, Fields: fields})
}

// writeServiceError maps domain errors onto HTTP status codes
func writeServiceError(w http.ResponseWriter, err error) {
	var vErr *model.ValidationError
	switch {
	case errors.As(err, &vErr):
		writeError(w, http.StatusBadRequest, "validation_error", "the request is invalid", vErr.Fields)
	case errors.Is(err, model.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error(), nil)
	case errors.Is(err, model.ErrRoomUnavailable):
		writeError(w, http.StatusConflict, "room_unavailable", err.Error(), nil)
	case errors.Is(err, model.ErrCapacityExceeded):
		writeError(w, http.StatusUnprocessableEntity, "capacity_exceeded", err.Error(), nil)
	default:
		slog.Error("unhandled error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred", nil)
	}
}

// decodeJSON reads a single JSON object, rejecting unknown fields and
// oversized bodies so typos in field names don't silently pass
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		return errors.New("Content-Type must be application/json")
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var typeErr *json.UnmarshalTypeError
		switch {
		case errors.Is(err, io.EOF):
			return errors.New("request body is empty")
		case errors.As(err, &syntaxErr):
			return fmt.Errorf("malformed JSON at position %d", syntaxErr.Offset)
		case errors.As(err, &typeErr):
			return fmt.Errorf("field %q has the wrong type", typeErr.Field)
		default:
			return err
		}
	}
	if dec.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
