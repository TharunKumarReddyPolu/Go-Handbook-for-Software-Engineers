package calc

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	buf := &bytes.Buffer{}
	return NewHandler(slog.New(slog.NewTextHandler(buf, nil)))
}

func postEval(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/eval", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder) EvalResponse {
	t.Helper()
	var resp EvalResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return resp
}

func TestEvalHandler_Created(t *testing.T) {
	rec := postEval(t, testHandler(t), `{"expr":"1+2"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	resp := decodeResponse(t, rec)
	if resp.Result != 3 {
		t.Errorf("result = %d, want 3", resp.Result)
	}
	if resp.Error != "" {
		t.Errorf("error = %q, want empty", resp.Error)
	}
}

func TestEvalHandler_SyntaxErrorIs400(t *testing.T) {
	rec := postEval(t, testHandler(t), `{"expr":"1+"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	if resp := decodeResponse(t, rec); resp.Error == "" {
		t.Error("syntax errors must carry a message")
	}
}

func TestEvalHandler_DivideByZeroIs400(t *testing.T) {
	rec := postEval(t, testHandler(t), `{"expr":"1/0"}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
	resp := decodeResponse(t, rec)
	if resp.Error != "division by zero" {
		t.Errorf("error = %q, want 'division by zero'", resp.Error)
	}
	// The internal sentinel must not leak into the response body.
	if bytes.Contains(rec.Body.Bytes(), []byte("divide")) {
		t.Error("response leaks internal error text")
	}
}

func TestEvalHandler_MalformedJSON(t *testing.T) {
	rec := postEval(t, testHandler(t), `{bad`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
