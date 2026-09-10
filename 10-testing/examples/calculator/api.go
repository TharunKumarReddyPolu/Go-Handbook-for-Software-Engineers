package calc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// EvalRequest is the wire input.
type EvalRequest struct {
	Expr string `json:"expr"`
}

// EvalResponse is the wire output; Error carries a safe message.
type EvalResponse struct {
	Result int    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// NewHandler returns the HTTP layer. It logs once per failed request and
// maps internal errors to safe responses: the boundary discipline from
// 05-errors applied to a tiny service.
func NewHandler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /eval", func(w http.ResponseWriter, r *http.Request) {
		var req EvalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.WarnContext(r.Context(), "bad json", "err", err)
			writeJSON(w, http.StatusBadRequest, EvalResponse{Error: "malformed JSON"})
			return
		}

		result, err := Eval(req.Expr)
		if err != nil {
			var se *ErrSyntax
			switch {
			case errors.As(err, &se):
				writeJSON(w, http.StatusBadRequest, EvalResponse{Error: err.Error()})
			case errors.Is(err, ErrDivideByZero):
				writeJSON(w, http.StatusBadRequest, EvalResponse{Error: "division by zero"})
			default:
				logger.ErrorContext(r.Context(), "eval failed", "err", err)
				writeJSON(w, http.StatusInternalServerError, EvalResponse{Error: "internal error"})
			}
			return
		}
		writeJSON(w, http.StatusOK, EvalResponse{Result: result})
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error(fmt.Sprintf("write response: %v", err))
	}
}
