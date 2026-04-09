package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pack-calc/pkg/services"
)

// Mock storage
type mockStorage struct {
	packs  []int
	setErr error
	getErr error
}

func (m *mockStorage) GetPacks() ([]int, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	cp := make([]int, len(m.packs))
	copy(cp, m.packs)
	return cp, nil
}

func (m *mockStorage) SetPacks(packs []int) ([]int, error) {
	if m.setErr != nil {
		return nil, m.setErr
	}
	m.packs = packs
	cp := make([]int, len(packs))
	copy(cp, packs)
	return cp, nil
}

func newTestHandler(s *mockStorage) *Handler {
	log := slog.Default()
	return NewHandler(s, log)
}

func newMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

// GET /api/v1/packs
func TestGetPacks(t *testing.T) {
	s := &mockStorage{packs: []int{5000, 2000, 1000, 500, 250}}
	mux := newMux(newTestHandler(s))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/packs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp packsResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Packs) != 5 {
		t.Fatalf("expected 5 packs, got %d", len(resp.Packs))
	}
}

func TestGetPacksStorageError(t *testing.T) {
	s := &mockStorage{getErr: errTest}
	mux := newMux(newTestHandler(s))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/packs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// POST /api/v1/packs
func TestSetPacks(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	body := `{"packs": [100, 500, 1000]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp packsResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Packs) != 3 {
		t.Fatalf("expected 3 packs, got %d", len(resp.Packs))
	}
}

func TestSetPacksInvalidJSON(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/packs", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSetPacksValidationError(t *testing.T) {
	s := &mockStorage{packs: []int{250}, setErr: errValidation}
	mux := newMux(newTestHandler(s))

	body := `{"packs": [-1]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// POST /api/v1/calculate
func TestCalculate(t *testing.T) {
	s := &mockStorage{packs: []int{5000, 2000, 1000, 500, 250}}
	mux := newMux(newTestHandler(s))

	body := `{"items": 501}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp calculateResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Packs["500"] != 1 || resp.Packs["250"] != 1 {
		t.Fatalf("expected {500:1, 250:1}, got %v", resp.Packs)
	}
	if len(resp.PackSizesUsed) != 5 {
		t.Fatalf("expected 5 pack sizes used, got %d", len(resp.PackSizesUsed))
	}
}

func TestCalculateZeroItems(t *testing.T) {
	s := &mockStorage{packs: []int{250, 500}}
	mux := newMux(newTestHandler(s))

	body := `{"items": 0}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp calculateResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp.Packs) != 0 {
		t.Fatalf("expected empty packs, got %v", resp.Packs)
	}
}

func TestCalculateNegativeItems(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	body := `{"items": -5}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCalculateInvalidJSON(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString("{bad"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCalculateStorageError(t *testing.T) {
	s := &mockStorage{getErr: errTest}
	mux := newMux(newTestHandler(s))

	body := `{"items": 10}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// Method not allowed
func TestWrongMethod(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/packs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		t.Fatal("expected non-200 for DELETE on /api/v1/packs")
	}
}

// Content-Type header
func TestResponseContentType(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/packs", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %q", ct)
	}
}

// CalcPacks returns error (empty packs from storage)
func TestCalculateCalcPacksError(t *testing.T) {
	s := &mockStorage{packs: []int{}}
	mux := newMux(newTestHandler(s))

	body := `{"items": 1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}

	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == "" {
		t.Fatal("expected non-empty error message")
	}
}

// Items exceeds upper bound
func TestCalculateItemsTooLarge(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	body := `{"items": 9999999999}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error == "" {
		t.Fatal("expected non-empty error message for items too large")
	}
}

// Request body too large
func TestSetPacksBodyTooLarge(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	// Create a body larger than 1 MB
	bigBody := `{"packs": [` + strings.Repeat("1,", 600000) + `1]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packs", bytes.NewBufferString(bigBody))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", w.Code)
	}
}

func TestCalculateBodyTooLarge(t *testing.T) {
	s := &mockStorage{packs: []int{250}}
	mux := newMux(newTestHandler(s))

	bigBody := `{"items": 1, "extra": "` + strings.Repeat("x", 2*1024*1024) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/calculate", bytes.NewBufferString(bigBody))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized body, got %d", w.Code)
	}
}

// Too many packs (rejected by storage)
func TestSetPacksTooMany(t *testing.T) {
	s := &mockStorage{packs: []int{250}, setErr: &testError{"too many pack sizes: 21 (max 20)"}}
	mux := newMux(newTestHandler(s))

	body := `{"packs": [1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "too many") {
		t.Fatalf("expected 'too many' in error, got %q", resp.Error)
	}
}

// Pack size too large (rejected by storage)
func TestSetPacksSizeTooLarge(t *testing.T) {
	s := &mockStorage{packs: []int{250}, setErr: &testError{"pack size 2000000 exceeds maximum 1000000"}}
	mux := newMux(newTestHandler(s))

	body := `{"packs": [2000000]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp errorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !strings.Contains(resp.Error, "exceeds maximum") {
		t.Fatalf("expected 'exceeds maximum' in error, got %q", resp.Error)
	}
}

// SetPacks internal (IO) error returns 500, not 400
func TestSetPacksInternalErrorReturns500(t *testing.T) {
	s := &mockStorage{
		packs:  []int{250},
		setErr: &services.InternalError{Err: fmt.Errorf("persist packs: permission denied")},
	}
	mux := newMux(newTestHandler(s))

	body := `{"packs": [100]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/packs", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for internal error, got %d", w.Code)
	}
}

// Sentinel errors for mock
var (
	errTest       = &testError{"storage error"}
	errValidation = &testError{"pack size must be positive, got -1"}
)

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
