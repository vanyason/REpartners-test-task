package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// PackStorage defines the contract for pack persistence
type PackStorage interface {
	GetPacks() ([]int, error)
	SetPacks(packs []int) ([]int, error)
}

type packFile struct {
	Packs []int `json:"packs"`
}

// FilePackStorage persists pack sizes to a JSON file with
// an in-memory cache protected by RWMutex
type FilePackStorage struct {
	mu       sync.RWMutex
	filePath string
	packs    []int
}

// NewFilePackStorage creates a FilePackStorage
// If the file exists, packs are loaded from it
// Otherwise defaultPacks are validated, written to the file, and cached
func NewFilePackStorage(filePath string, defaultPacks []int) (*FilePackStorage, error) {
	s := &FilePackStorage{filePath: filePath}

	data, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		cleaned, vErr := validateAndClean(defaultPacks)
		if vErr != nil {
			return nil, fmt.Errorf("invalid default packs: %w", vErr)
		}
		s.packs = cleaned
		if wErr := s.writeLocked(); wErr != nil {
			return nil, fmt.Errorf("write default packs: %w", wErr)
		}
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read packs file: %w", err)
	}

	var pf packFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parse packs file: %w", err)
	}

	cleaned, err := validateAndClean(pf.Packs)
	if err != nil {
		return nil, fmt.Errorf("invalid packs in file: %w", err)
	}
	s.packs = cleaned
	return s, nil
}

func (s *FilePackStorage) GetPacks() ([]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.packs), nil
}

func (s *FilePackStorage) SetPacks(packs []int) ([]int, error) {
	cleaned, err := validateAndClean(packs)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.packs = cleaned
	if err := s.writeLocked(); err != nil {
		return nil, fmt.Errorf("persist packs: %w", err)
	}
	return slices.Clone(s.packs), nil
}

// writeLocked writes packs to disk, caller must hold s.mu
func (s *FilePackStorage) writeLocked() error {
	data, err := json.Marshal(packFile{Packs: s.packs})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(s.filePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0o644)
}

// Normalize packs
func validateAndClean(packs []int) ([]int, error) {
	if len(packs) == 0 {
		return nil, errors.New("packs must not be empty")
	}

	seen := make(map[int]struct{}, len(packs))
	cleaned := make([]int, 0, len(packs))
	for _, p := range packs {
		if p <= 0 {
			return nil, fmt.Errorf("pack size must be positive, got %d", p)
		}
		if _, dup := seen[p]; !dup {
			seen[p] = struct{}{}
			cleaned = append(cleaned, p)
		}
	}

	slices.SortFunc(cleaned, func(a, b int) int { return b - a }) // descending
	return cleaned, nil
}
