package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"pack-calc/pkg/services"
)

// statusWriter tests
func TestStatusWriterDefault(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, status: http.StatusOK}

	// Write body without calling WriteHeader
	sw.Write([]byte("hello"))

	if sw.status != http.StatusOK {
		t.Fatalf("expected default status 200, got %d", sw.status)
	}
	if rec.Body.String() != "hello" {
		t.Fatalf("expected body 'hello', got %q", rec.Body.String())
	}
}

func TestStatusWriterWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, status: http.StatusOK}

	sw.WriteHeader(http.StatusNotFound)

	if sw.status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", sw.status)
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected underlying recorder status 404, got %d", rec.Code)
	}
}

// loggingMiddleware tests
func TestLoggingMiddlewarePassesThrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	handler := loggingMiddleware(log, inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected body 'ok', got %q", rec.Body.String())
	}
}

func TestLoggingMiddlewareNon200(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad"))
	})

	log := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	handler := loggingMiddleware(log, inner)

	req := httptest.NewRequest(http.MethodPost, "/fail", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLoggingMiddlewareLogs(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	handler := loggingMiddleware(log, inner)

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d: %s", len(lines), output)
	}

	// Verify request log
	var reqLog map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &reqLog); err != nil {
		t.Fatalf("failed to parse request log: %v", err)
	}
	if reqLog["msg"] != "request" {
		t.Fatalf("expected msg 'request', got %v", reqLog["msg"])
	}
	if reqLog["method"] != "GET" {
		t.Fatalf("expected method GET, got %v", reqLog["method"])
	}
	if reqLog["path"] != "/api/test" {
		t.Fatalf("expected path '/api/test', got %v", reqLog["path"])
	}

	// Verify response log
	var respLog map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &respLog); err != nil {
		t.Fatalf("failed to parse response log: %v", err)
	}
	if respLog["msg"] != "response" {
		t.Fatalf("expected msg 'response', got %v", respLog["msg"])
	}
	if respLog["method"] != "GET" {
		t.Fatalf("expected method GET, got %v", respLog["method"])
	}
	if int(respLog["status"].(float64)) != http.StatusCreated {
		t.Fatalf("expected status 201, got %v", respLog["status"])
	}
	if _, ok := respLog["duration"]; !ok {
		t.Fatal("expected duration field in response log")
	}
}

// buildServer test
type testStorage struct {
	packs []int
}

func (s *testStorage) GetPacks() ([]int, error) {
	cp := make([]int, len(s.packs))
	copy(cp, s.packs)
	return cp, nil
}

func (s *testStorage) SetPacks(packs []int) ([]int, error) {
	s.packs = packs
	cp := make([]int, len(packs))
	copy(cp, packs)
	return cp, nil
}

func TestBuildServer(t *testing.T) {
	storage := &testStorage{packs: []int{250, 500}}
	log := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))

	srv := buildServer(storage, log, ":0")

	if srv.Addr != ":0" {
		t.Fatalf("expected addr ':0', got %q", srv.Addr)
	}
	if srv.Handler == nil {
		t.Fatal("expected non-nil handler")
	}
	if srv.ReadTimeout != 5_000_000_000 {
		t.Fatalf("expected ReadTimeout 5s, got %v", srv.ReadTimeout)
	}
	if srv.WriteTimeout != 10_000_000_000 {
		t.Fatalf("expected WriteTimeout 10s, got %v", srv.WriteTimeout)
	}
	if srv.IdleTimeout != 60_000_000_000 {
		t.Fatalf("expected IdleTimeout 60s, got %v", srv.IdleTimeout)
	}
}

func TestBuildServerRoutesWork(t *testing.T) {
	fp := t.TempDir() + "/packs.json"
	storage, err := services.NewFilePackStorage(fp, []int{250, 500, 1000})
	if err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	srv := buildServer(storage, log, ":0")

	// Test GET /api/v1/packs through the full server handler
	req := httptest.NewRequest(http.MethodGet, "/api/v1/packs", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Test static file serving
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for static root, got %d", rec.Code)
	}
}
