package services

import (
	"testing"
)

var defaultPacks = []int{250, 500, 1000, 2000, 5000}

// Helper
func assertPacksEqual(t *testing.T, expected, got PacksCount) {
	t.Helper()
	if len(expected) != len(got) {
		t.Fatalf("expected %v, got %v", expected, got)
	}
	for k, v := range expected {
		if got[k] != v {
			t.Fatalf("expected %v, got %v", expected, got)
		}
	}
}

func TestPDFExamples(t *testing.T) {
	tests := []struct {
		name     string
		items    int
		expected PacksCount
	}{
		{"1 item", 1, PacksCount{250: 1}},
		{"250 items", 250, PacksCount{250: 1}},
		{"251 items", 251, PacksCount{500: 1}},
		{"501 items", 501, PacksCount{500: 1, 250: 1}},
		{"12001 items", 12001, PacksCount{5000: 2, 2000: 1, 250: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcPacks(tt.items, defaultPacks)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertPacksEqual(t, tt.expected, got)
		})
	}
}

// Edge case
func TestEdgeCase(t *testing.T) {
	packs := []int{23, 31, 53}
	got, err := CalcPacks(500000, packs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := PacksCount{53: 9429, 31: 7, 23: 2}
	assertPacksEqual(t, expected, got)
}

// Boundary cases
func TestZeroItems(t *testing.T) {
	got, err := CalcPacks(0, defaultPacks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestNegativeItems(t *testing.T) {
	_, err := CalcPacks(-1, defaultPacks)
	if err == nil {
		t.Fatal("expected error for negative items")
	}
}

func TestEmptyPacks(t *testing.T) {
	_, err := CalcPacks(100, []int{})
	if err == nil {
		t.Fatal("expected error for empty packs")
	}
}

func TestNilPacks(t *testing.T) {
	_, err := CalcPacks(100, nil)
	if err == nil {
		t.Fatal("expected error for nil packs")
	}
}

func TestPackSizeZero(t *testing.T) {
	_, err := CalcPacks(100, []int{0, 250})
	if err == nil {
		t.Fatal("expected error for pack size <= 0")
	}
}

func TestPackSizeNegative(t *testing.T) {
	_, err := CalcPacks(100, []int{-5, 250})
	if err == nil {
		t.Fatal("expected error for pack size <= 0")
	}
}

// Exact multiples
func TestExactMultiples(t *testing.T) {
	tests := []struct {
		name     string
		items    int
		expected PacksCount
	}{
		{"500 items", 500, PacksCount{500: 1}},
		{"10000 items", 10000, PacksCount{5000: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcPacks(tt.items, defaultPacks)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertPacksEqual(t, tt.expected, got)
		})
	}
}

// Single pack size
func TestSinglePackSize(t *testing.T) {
	tests := []struct {
		name     string
		items    int
		packs    []int
		expected PacksCount
	}{
		{"1 item with [250]", 1, []int{250}, PacksCount{250: 1}},
		{"251 items with [250]", 251, []int{250}, PacksCount{250: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcPacks(tt.items, tt.packs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertPacksEqual(t, tt.expected, got)
		})
	}
}

// Pack of 1 (performance)
func TestPackOfOne(t *testing.T) {
	got, err := CalcPacks(1_000_000, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := PacksCount{1: 1_000_000}
	assertPacksEqual(t, expected, got)
}

func TestPackOfOneWithOthers(t *testing.T) {
	got, err := CalcPacks(999, []int{1, 500, 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := PacksCount{500: 1, 1: 499}
	assertPacksEqual(t, expected, got)
}

// Rule 3 tiebreaker (same items, fewer packs)
func TestFewerPacks(t *testing.T) {
	tests := []struct {
		name     string
		items    int
		packs    []int
		expected PacksCount
	}{
		{"751 with [250,500,1000]", 751, []int{250, 500, 1000}, PacksCount{1000: 1}},
		{"501 with [250,500]", 501, []int{250, 500}, PacksCount{500: 1, 250: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcPacks(tt.items, tt.packs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertPacksEqual(t, tt.expected, got)
		})
	}
}

// Non-standard packs
func TestNonStandardPacks(t *testing.T) {
	tests := []struct {
		name     string
		items    int
		packs    []int
		expected PacksCount
	}{
		{"7 with [3,5]", 7, []int{3, 5}, PacksCount{5: 1, 3: 1}},
		{"6 with [3,5]", 6, []int{3, 5}, PacksCount{3: 2}},
		{"14 with [3,5]", 14, []int{3, 5}, PacksCount{5: 1, 3: 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalcPacks(tt.items, tt.packs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertPacksEqual(t, tt.expected, got)
		})
	}
}

// Non-coprime packs (GCD > 1)
func TestNonCoprimePacks(t *testing.T) {
	got, err := CalcPacks(5, []int{4, 6})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := PacksCount{6: 1}
	assertPacksEqual(t, expected, got)
}

// Large amounts
func TestLargeAmount(t *testing.T) {
	got, err := CalcPacks(999999, defaultPacks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := PacksCount{5000: 200}
	assertPacksEqual(t, expected, got)
}

// Robustness: unsorted and duplicate packs
func TestUnsortedPacks(t *testing.T) {
	got, err := CalcPacks(501, []int{5000, 250, 2000, 500, 1000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := PacksCount{500: 1, 250: 1}
	assertPacksEqual(t, expected, got)
}

func TestDuplicatePacks(t *testing.T) {
	got, err := CalcPacks(501, []int{250, 500, 500, 1000, 2000, 5000, 250})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := PacksCount{500: 1, 250: 1}
	assertPacksEqual(t, expected, got)
}

// All invalid packs
func TestAllInvalidPacks(t *testing.T) {
	_, err := CalcPacks(100, []int{0, -5})
	if err == nil {
		t.Fatal("expected error when all pack sizes are invalid")
	}
}

// Pack of 1 exact
func TestPackOfOneExact(t *testing.T) {
	got, err := CalcPacks(1, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPacksEqual(t, PacksCount{1: 1}, got)
}

// Frobenius gap overshoot
func TestFrobeniusGapOvershoot(t *testing.T) {
	got, err := CalcPacks(1, []int{3, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPacksEqual(t, PacksCount{3: 1}, got)
}

// Two equal packs (dedup)
func TestTwoEqualPacks(t *testing.T) {
	got, err := CalcPacks(10, []int{5, 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPacksEqual(t, PacksCount{5: 2}, got)
}

// Large single pack (pack >> items)
func TestLargeSinglePack(t *testing.T) {
	got, err := CalcPacks(1, []int{1000000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPacksEqual(t, PacksCount{1000000: 1}, got)
}

// Exact match on non-default packs
func TestExactMatchNonDefault(t *testing.T) {
	got, err := CalcPacks(53, []int{23, 31, 53})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPacksEqual(t, PacksCount{53: 1}, got)
}

// Pack of 1, million items (plan says 1B)
func TestPackOfOneMillion(t *testing.T) {
	got, err := CalcPacks(1_000_000, []int{1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPacksEqual(t, PacksCount{1: 1_000_000}, got)
}

// Benchmarks
func BenchmarkLargeOrderPrimes(b *testing.B) {
	packs := []int{23, 31, 53}
	for b.Loop() {
		CalcPacks(1_000_000_000, packs)
	}
}

func BenchmarkLargeOrderWithOne(b *testing.B) {
	packs := []int{1, 1000}
	for b.Loop() {
		CalcPacks(1_000_000_000, packs)
	}
}

func BenchmarkDefaultPacks(b *testing.B) {
	for b.Loop() {
		CalcPacks(12001, defaultPacks)
	}
}

func BenchmarkSinglePack(b *testing.B) {
	packs := []int{250}
	for b.Loop() {
		CalcPacks(1_000_000_000, packs)
	}
}
