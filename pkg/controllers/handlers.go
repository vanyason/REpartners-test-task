package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"pack-calc/pkg/services"
)

type Handler struct {
	storage services.PackStorage
	log     *slog.Logger
}

func NewHandler(storage services.PackStorage, log *slog.Logger) *Handler {
	return &Handler{storage: storage, log: log}
}

// RegisterRoutes wires all API routes onto the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/packs", h.GetPacks)
	mux.HandleFunc("POST /api/v1/packs", h.SetPacks)
	mux.HandleFunc("POST /api/v1/calculate", h.Calculate)
}

const maxBodySize = 1 << 20 // maxBodySize limits request body to 1 MB to prevent resource exhaustion

// Request / Response types
type packsResponse struct {
	Packs []int `json:"packs"`
}

type setPacksRequest struct {
	Packs []int `json:"packs"`
}

type calculateRequest struct {
	Items int `json:"items"`
}

type calculateResponse struct {
	Packs         map[string]int `json:"packs"`
	PackSizesUsed []int          `json:"pack_sizes_used"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// Handlers
func (h *Handler) GetPacks(w http.ResponseWriter, r *http.Request) {
	packs, err := h.storage.GetPacks()
	if err != nil {
		h.serverError(w, r, "get packs", err)
		return
	}

	writeJSON(w, http.StatusOK, packsResponse{Packs: packs})
}

func (h *Handler) SetPacks(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req setPacksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	saved, err := h.storage.SetPacks(req.Packs)
	if err != nil {
		var ie *services.InternalError
		if errors.As(err, &ie) {
			h.serverError(w, r, "set packs", err)
		} else {
			h.log.Warn("set packs rejected", "error", err)
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, packsResponse{Packs: saved})
}

func (h *Handler) Calculate(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	var req calculateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	if req.Items < 0 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "items must be non-negative"})
		return
	}
	if req.Items > services.MaxItems {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: fmt.Sprintf("items must be at most %d", services.MaxItems),
		})
		return
	}

	packs, err := h.storage.GetPacks()
	if err != nil {
		h.serverError(w, r, "get packs for calculate", err)
		return
	}

	result, err := services.CalcPacks(req.Items, packs)
	if err != nil {
		var ie *services.InternalError
		if errors.As(err, &ie) {
			h.serverError(w, r, "calculation", err)
		} else {
			h.log.Warn("calculation failed", "items", req.Items, "error", err)
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		}
		return
	}

	// Convert int keys to string keys for JSON
	strResult := make(map[string]int, len(result))
	for k, v := range result {
		strResult[strconv.Itoa(k)] = v
	}

	writeJSON(w, http.StatusOK, calculateResponse{
		Packs:         strResult,
		PackSizesUsed: packs,
	})
}

// Helpers
func (h *Handler) serverError(w http.ResponseWriter, r *http.Request, context string, err error) {
	h.log.Error(context, "error", err, "method", r.Method, "path", r.URL.Path)
	writeJSON(w, http.StatusInternalServerError, errorResponse{
		Error: fmt.Sprintf("internal error: %s", context),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}
