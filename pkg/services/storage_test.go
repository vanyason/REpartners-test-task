package services

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func tempFilePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "packs.json")
}

func TestStorageCreatesFileWithDefaults(t *testing.T) {
	fp := tempFilePath(t)
	s, err := NewFilePackStorage(fp, []int{500, 250, 1000})
	if err != nil {
		t.Fatal(err)
	}

	packs, _ := s.GetPacks()
	assertSliceEqual(t, []int{1000, 500, 250}, packs)

	if _, err := os.Stat(fp); err != nil {
		t.Fatalf("file not created: %v", err)
	}
}

func TestStorageLoadsExistingFile(t *testing.T) {
	fp := tempFilePath(t)
	os.WriteFile(fp, []byte(`{"packs":[300,100,200]}`), 0o644)

	s, err := NewFilePackStorage(fp, nil)
	if err != nil {
		t.Fatal(err)
	}

	packs, _ := s.GetPacks()
	assertSliceEqual(t, []int{300, 200, 100}, packs)
}

func TestStorageCorruptFile(t *testing.T) {
	fp := tempFilePath(t)
	os.WriteFile(fp, []byte(`not json`), 0o644)

	_, err := NewFilePackStorage(fp, nil)
	if err == nil {
		t.Fatal("expected error for corrupt file")
	}
}

func TestStorageInvalidDefaults(t *testing.T) {
	fp := tempFilePath(t)
	_, err := NewFilePackStorage(fp, []int{})
	if err == nil {
		t.Fatal("expected error for empty defaults")
	}

	_, err = NewFilePackStorage(fp, []int{-1})
	if err == nil {
		t.Fatal("expected error for negative default")
	}
}

func TestStorageGetPacksReturnsCopy(t *testing.T) {
	fp := tempFilePath(t)
	s, _ := NewFilePackStorage(fp, []int{100, 200})

	packs, _ := s.GetPacks()
	packs[0] = 9999

	packs2, _ := s.GetPacks()
	if packs2[0] == 9999 {
		t.Fatal("GetPacks must return a copy, not the internal slice")
	}
}

func TestStorageSetPacks(t *testing.T) {
	fp := tempFilePath(t)
	s, _ := NewFilePackStorage(fp, []int{100})

	got, err := s.SetPacks([]int{50, 300, 100})
	if err != nil {
		t.Fatal(err)
	}
	assertSliceEqual(t, []int{300, 100, 50}, got)

	// Verify persistence by loading fresh
	s2, _ := NewFilePackStorage(fp, nil)
	packs, _ := s2.GetPacks()
	assertSliceEqual(t, []int{300, 100, 50}, packs)
}

func TestStorageSetPacksDeduplicates(t *testing.T) {
	fp := tempFilePath(t)
	s, _ := NewFilePackStorage(fp, []int{100})

	got, err := s.SetPacks([]int{50, 50, 100, 100})
	if err != nil {
		t.Fatal(err)
	}
	assertSliceEqual(t, []int{100, 50}, got)
}

func TestStorageSetPacksValidation(t *testing.T) {
	fp := tempFilePath(t)
	s, _ := NewFilePackStorage(fp, []int{100})

	if _, err := s.SetPacks([]int{}); err == nil {
		t.Fatal("expected error for empty packs")
	}
	if _, err := s.SetPacks([]int{0}); err == nil {
		t.Fatal("expected error for zero pack")
	}
	if _, err := s.SetPacks([]int{-5}); err == nil {
		t.Fatal("expected error for negative pack")
	}

	packs, _ := s.GetPacks()
	assertSliceEqual(t, []int{100}, packs)
}

func TestStorageConcurrentAccess(t *testing.T) {
	fp := tempFilePath(t)
	s, _ := NewFilePackStorage(fp, []int{100, 200, 300})

	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.GetPacks()
		}()
		go func() {
			defer wg.Done()
			s.SetPacks([]int{100, 200, 300})
		}()
	}
	wg.Wait()
}

func TestStorageSetPacksWriteError(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "packs.json")
	s, err := NewFilePackStorage(fp, []int{100, 200})
	if err != nil {
		t.Fatal(err)
	}

	// Make directory read-only so writeLocked fails
	os.Chmod(dir, 0o444)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	// Guard for running as root
	if f, err := os.Create(filepath.Join(dir, "probe")); err == nil {
		f.Close()
		t.Skip("running as root, cannot test permission errors")
	}

	_, err = s.SetPacks([]int{300})
	if err == nil {
		t.Fatal("expected error when directory is read-only")
	}
}

func TestStorageNewReadError(t *testing.T) {
	fp := tempFilePath(t)
	os.WriteFile(fp, []byte(`{"packs":[100]}`), 0o644)

	// Make file unreadable
	os.Chmod(fp, 0o000)
	t.Cleanup(func() { os.Chmod(fp, 0o644) })

	// Guard for running as root
	if _, err := os.ReadFile(fp); err == nil {
		t.Skip("running as root, cannot test permission errors")
	}

	_, err := NewFilePackStorage(fp, nil)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}
}

func TestStorageNewWriteDefaultsError(t *testing.T) {
	dir := t.TempDir()
	// Make directory read-only so file creation fails
	os.Chmod(dir, 0o444)
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	// Guard for running as root
	fp := filepath.Join(dir, "probe")
	if f, err := os.Create(fp); err == nil {
		f.Close()
		t.Skip("running as root, cannot test permission errors")
	}

	_, err := NewFilePackStorage(filepath.Join(dir, "packs.json"), []int{100})
	if err == nil {
		t.Fatal("expected error when directory is read-only")
	}
}

func TestStorageNewInvalidPacksInFile(t *testing.T) {
	fp := tempFilePath(t)
	os.WriteFile(fp, []byte(`{"packs":[0,-1]}`), 0o644)

	_, err := NewFilePackStorage(fp, nil)
	if err == nil {
		t.Fatal("expected error for invalid packs in file")
	}
}

func assertSliceEqual(t *testing.T, expected, got []int) {
	t.Helper()
	if len(expected) != len(got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	for i := range expected {
		if expected[i] != got[i] {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	}
}
