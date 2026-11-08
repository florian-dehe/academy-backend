// Package httpx holds HTTP helpers: typed errors, JSON IO, middleware.
package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/render"
)

// Error is a typed domain/HTTP error. Domain code returns these (or wraps them);
// WriteError maps them centrally.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message,omitempty"`
}

func (e *Error) Error() string { return e.Code }

var (
	ErrBadRequest           = &Error{Status: http.StatusBadRequest, Code: "bad_request"}
	ErrUnauthorized         = &Error{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication required"}
	ErrSubscriptionRequired = &Error{Status: http.StatusPaymentRequired, Code: "subscription_required"}
	ErrForbidden            = &Error{Status: http.StatusForbidden, Code: "forbidden"}
	ErrNotFound             = &Error{Status: http.StatusNotFound, Code: "not_found"}
	ErrMethodNotAllowed     = &Error{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed"}
	ErrVersionNotDraft      = &Error{Status: http.StatusConflict, Code: "version_not_draft"}
	ErrUserRequired         = &Error{Status: http.StatusForbidden, Code: "user_required", Message: "academy user required"}
	ErrConflict             = &Error{Status: http.StatusConflict, Code: "conflict"}
	ErrUnavailable          = &Error{Status: http.StatusServiceUnavailable, Code: "upstream_unavailable", Message: "authentication service unavailable"}
	ErrNotImplemented       = &Error{Status: http.StatusNotImplemented, Code: "not_implemented"}
)

// WithMessage returns a copy of e with a custom message.
func (e *Error) WithMessage(msg string) *Error {
	c := *e
	c.Message = msg
	return &c
}

type errorBody struct {
	Error *Error `json:"error"`
}

func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	var he *Error
	if !errors.As(err, &he) {
		slog.ErrorContext(r.Context(), "unhandled error", "err", err, "path", r.URL.Path)
		he = &Error{Status: http.StatusInternalServerError, Code: "internal", Message: "internal error"}
	}
	render.Status(r, he.Status)
	render.JSON(w, r, errorBody{Error: he})
}

// HandlerFunc is a handler that returns an error for central mapping.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handle adapts a HandlerFunc to http.HandlerFunc.
func Handle(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			WriteError(w, r, err)
		}
	}
}

// NotImplemented is the placeholder for routes not built yet.
func NotImplemented(_ http.ResponseWriter, _ *http.Request) error { return ErrNotImplemented }

// NotFound and MethodNotAllowed give unmatched routes the standard JSON error body.
func NotFound(w http.ResponseWriter, r *http.Request) { WriteError(w, r, ErrNotFound) }

func MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	WriteError(w, r, ErrMethodNotAllowed)
}

// Recoverer turns a handler panic into a logged 500 with the standard JSON body.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler { //nolint:errorlint // sentinel is compared by identity
					panic(v)
				}
				slog.ErrorContext(ctx, "panic", "value", v, "path", r.URL.Path)
				WriteError(w, r, errors.New("panic"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// render.Bind decodes through render.Decode, which by default accepts any
// size and unknown fields. Route it through the strict Decode below, so
// handlers can use chi's render.Bind idiom with our limits.
func init() { render.Decode = Decode }

// MaxBodyBytes caps JSON request bodies read by Decode.
const MaxBodyBytes = 1 << 20

// Decode reads one JSON value (max MaxBodyBytes) into v, rejecting unknown
// fields and trailing data.
func Decode(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return ErrBadRequest.WithMessage("invalid JSON body: " + err.Error())
	}
	if dec.More() {
		return ErrBadRequest.WithMessage("invalid JSON body: trailing data")
	}
	return nil
}
