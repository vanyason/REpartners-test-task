package main

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pack-calc/pkg/controllers"
	"pack-calc/pkg/services"
	"pack-calc/pkg/static"
)

const (
	defaultAddr     = ":8080"
	defaultPackFile = "data/packs.json"
	shutdownTimeout = 5 * time.Second
)

var defaultPacks = []int{250, 500, 1000, 2000, 5000}

func main() {
	// Init slog
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(log)

	// Optionally set port and filename
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	packFile := os.Getenv("PACK_FILE")
	if packFile == "" {
		packFile = defaultPackFile
	}

	// Init
	storage, err := services.NewFilePackStorage(packFile, defaultPacks)
	if err != nil {
		log.Error("failed to init pack storage", "error", err)
		os.Exit(1)
	}

	srv := buildServer(storage, log, addr)

	// Graceful shutdown
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		log.Info("shutting down", "signal", sig)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Error("shutdown error", "error", err)
		}
		close(done)
	}()

	// Start server
	log.Info("server starting", "addr", addr, "pack_file", packFile)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Error("server error", "error", err)
		os.Exit(1)
	}

	<-done
	log.Info("server stopped")
}

// buildServer creates and configures the HTTP server with all routes and middleware
func buildServer(storage services.PackStorage, log *slog.Logger, addr string) *http.Server {
	handler := controllers.NewHandler(storage, log)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Serve embedded frontend (HTML, CSS, JS)
	staticFS, _ := fs.Sub(static.Files, ".")
	mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	return &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(log, mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

// Log all the requests via middleware
func loggingMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
		)
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Info("response",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start).String(),
		)
	})
}

// Custom wrapper to save response status code for loggingMiddleware
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}
