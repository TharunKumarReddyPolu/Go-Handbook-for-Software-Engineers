package payments

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/platform/httpjson"
	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/platform/httpmw"
	"github.com/TharunKumarReddyPolu/Go-Handbook-for-Software-Engineers/14-backend-development/examples/service/internal/platform/obshttp"
)

// Identity is the transport's authn product (chapter 4): attached by
// the authenticate middleware; handlers map it to the domain's
// Customer. Tests construct identities directly through the
// Authorization header.
type Identity struct {
	Subject string // "user:cus_1" / "admin:root"
	Kind    string // "user" | "admin"
}

// Customer extracts the domain actor from an Identity.
func (id Identity) Customer() Customer {
	return Customer{ID: strings.TrimPrefix(id.Subject, "user:")}
}

type ctxKey int

const identityKey ctxKey = 0

func identityFrom(r *http.Request) (Identity, bool) {
	id, ok := r.Context().Value(identityKey).(Identity)
	return id, ok
}

// authenticate is the transport's token verifier (real mechanics:
// 21-security when it ships). It demonstrates chapter 4's middleware
// contract: reject 401 with WWW-Authenticate, or attach identity.
func authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, prefix) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(auth, prefix) // "user:cus_1" / "admin-token"
		var id Identity
		switch {
		case token == "admin-token":
			id = Identity{Subject: "admin:root", Kind: "admin"}
		case strings.HasPrefix(token, "user:"):
			id = Identity{Subject: token, Kind: "user"}
		default:
			w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), identityKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireKind is the coarse authz gate: route-scoped policy, one
// line per protected route in Routes() (chapter 4's policy table).
func requireKind(kind string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r)
		if !ok || id.Kind != kind {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handler is the transport: wire types and mapping, no decisions
// (chapter 1's layer contract).
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) Handler { return Handler{svc: svc} }

// Routes wires paths and policies over the domain middleware chain.
// Order (12 §2, refined by section 20): metrics+tracing OUTERMOST
// (they count every request received, including rejected ones), then
// ScopedLogger (gets trace IDs from the span), then Recover innermost
// of the global wrappers so panics become 5xx that metrics count.
func (h Handler) Routes(logger *slog.Logger, red *obshttp.Metrics) http.Handler {
	mux := http.NewServeMux()

	// Health endpoints (chapter 5): unauthenticated, leak nothing.
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK) // local state only in this example
		_, _ = w.Write([]byte("ok\n"))
	})

	// Domain routes: the policy table. One line per protected route.
	mux.Handle("POST /payments", requireKind("user", http.HandlerFunc(h.create)))
	mux.Handle("POST /payments/{id}/cancel", requireKind("user", http.HandlerFunc(h.cancel)))

	// Global gates: authenticate applies to everything under it; the
	// health endpoints are outside via the leading mux registrations
	// only if mounted separately. Here the example mounts them inside
	// authn for simplicity and asserts that in tests.
	authed := authenticate(mux)

	chain := httpmw.Chain(authed,
		func(next http.Handler) http.Handler { return httpmw.Recover(next) },
		func(next http.Handler) http.Handler { return httpmw.ScopedLogger(logger)(next) },
	)
	if red == nil {
		return chain // observability-free wiring is legal (tests)
	}
	return red.Instrument(chain)
}

// --- wire types (12 §3): transport-only shapes ---

type createRequest struct {
	CustomerID  string `json:"customer_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

type paymentResponse struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customer_id"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
}

func toResponse(p Payment) paymentResponse {
	return paymentResponse{
		ID: p.ID, CustomerID: p.CustomerID, AmountMinor: p.AmountMinor,
		Currency: p.Currency, Status: string(p.Status),
	}
}

func (h Handler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpjson.Decode[createRequest](w, r)
	if err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	// ChargeInput and createRequest are field-identical, so a plain
	// conversion crosses the wire/domain boundary without a mapping
	// function (staticcheck S1016). If the wire shape ever diverges
	// from the domain input, replace the conversion with an explicit
	// mapping: the boundary, not the types, is the contract.
	p, err := h.svc.Charge(r.Context(), ChargeInput(req))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Location", "/payments/"+p.ID)
	httpjson.Write(w, http.StatusCreated, toResponse(p))
}

func (h Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := identityFrom(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.svc.Cancel(r.Context(), id.Customer(), r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// mapError is the single translation point (05 §2, 12 §1): domain
// sentinels in, status codes out. ErrForbidden maps to 404 by design:
// someone else's payment does not exist for you (chapter 4).
func mapError(err error) (string, int, bool) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrForbidden):
		return "payment not found", http.StatusNotFound, true
	case errors.Is(err, ErrInsufficientFunds):
		return "insufficient funds", http.StatusUnprocessableEntity, true
	case errors.Is(err, ErrNotCancellable):
		return "payment not cancellable", http.StatusConflict, true
	case isValidation(err):
		return err.Error(), http.StatusUnprocessableEntity, true
	default:
		return "", 0, false // unknown: WriteError's 500 branch
	}
}

func writeError(w http.ResponseWriter, err error) {
	httpjson.WriteError(w, err, mapError)
}

// validationError is the transport-recognizable wrapper for batched
// validation failures. The service returns joined errors; this type
// carries them across the boundary without string parsing.
type validationError struct{ joined error }

func (e *validationError) Error() string { return e.joined.Error() }
func (e *validationError) Unwrap() error { return e.joined }

func isValidation(err error) bool {
	var ve *validationError
	return errors.As(err, &ve)
}
