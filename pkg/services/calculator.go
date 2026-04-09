package services

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

const maxDPSize = 1_000_000 // caps the DP table to prevent excessive memory allocation (≈ 24 MB)

// dpEntry stores the best way to fill exactly `t` items in the DP table
type dpEntry struct {
	totalPacks int  // minimum number of packs needed
	lastPack   int  // which pack size was used last (for backtracking the solution)
	valid      bool // whether this amount is achievable at all
}

// CalcPacks finds the minimum combination of packs to fulfill an order of `items` units
//
// The goal: ship at least `items` units using the fewest packs possible,
// while also minimizing overshipping (sending more items than ordered)
//
// Algorithm overview (example: 263 items, packs [250, 500, 1000, 2000, 5000]):
//
//  1. Validate & deduplicate pack sizes, sort descending: [5000, 2000, 1000, 500, 250]
//
//  2. Compute GCD of all pack sizes (here GCD = 250) and round the target UP
//     to the nearest multiple of GCD: 263 → 500. This is because we can only
//     ever ship in multiples of the GCD, so the true minimum is 500 items
//
//  3. Divide everything by the GCD to work with smaller numbers:
//     packs become [20, 8, 4, 2, 1], target becomes 2
//
//  4. Build a DP (dynamic programming) table for small amounts. For each amount
//     from 1..dpSize, find the fewest packs to reach it exactly
//
//  5. Search for the optimal answer by trying different quantities of the largest
//     pack (q = how many "big" packs to use), then looking up the remainder in the
//     DP table. We pick the combination that ships the fewest extra items, breaking
//     ties by fewest total packs
//     For our example: target=2 in reduced units → 0 big packs (size 20) + DP[2] = 2×1
//     → back in real units: 2 × 250 = 500 items total → result: {250: 2}
//
//  6. Reconstruct which packs were used by backtracking through the DP table,
//     then scale pack sizes back up by the GCD
func CalcPacks(items int, packs []int) (PacksCount, error) {
	if items < 0 {
		return nil, errors.New("items must be non-negative")
	}
	if items == 0 {
		return PacksCount{}, nil
	}
	if items > MaxItems {
		return nil, fmt.Errorf("items must be at most %d", MaxItems)
	}
	if len(packs) > MaxPackCount {
		return nil, fmt.Errorf("too many pack sizes: %d (max %d)", len(packs), MaxPackCount)
	}

	// Step 1: Deduplicate, validate, and sort pack sizes descending
	seen := make(map[int]bool)
	var cleaned []int
	for _, p := range packs {
		if p <= 0 {
			return nil, errors.New("pack size must be positive")
		}
		if p > MaxPackSize {
			return nil, fmt.Errorf("pack size %d exceeds maximum %d", p, MaxPackSize)
		}
		if !seen[p] {
			seen[p] = true
			cleaned = append(cleaned, p)
		}
	}
	if len(cleaned) == 0 {
		return nil, errors.New("no valid pack sizes")
	}

	slices.SortFunc(cleaned, func(a, b int) int { return cmp.Compare(b, a) })

	// Special case: only one pack size → just round up with ceil division
	if len(cleaned) == 1 {
		p := cleaned[0]
		count := (items + p - 1) / p
		return PacksCount{p: count}, nil
	}

	// Step 2: Compute GCD and round the target up
	// Any achievable total is a multiple of GCD, so we snap the target upward
	g := cleaned[0]
	for _, p := range cleaned[1:] {
		g = gcd(g, p)
	}
	target := ((items + g - 1) / g) * g

	// Step 3: Divide pack sizes and target by GCD to reduce the problem
	// This makes numbers smaller and the DP table more compact
	reduced := make([]int, len(cleaned))
	for i, p := range cleaned {
		reduced[i] = p / g
	}
	rTarget := target / g

	// Step 4: Build the DP table
	// dp[t] = fewest packs to fill exactly t (reduced) items
	// Table size is bounded by the Frobenius-number estimate (smallest × second-smallest),
	// which guarantees all larger amounts are representable
	smallest := reduced[len(reduced)-1]
	secondSmallest := reduced[len(reduced)-2]
	dpSize := smallest * secondSmallest // Guard against integer overflow: check before multiplying
	if dpSize/smallest != secondSmallest {
		dpSize = maxDPSize // overflow — clamp
	}
	dpSize = min(max(dpSize, reduced[0]), maxDPSize)

	dp := make([]dpEntry, dpSize+1)
	dp[0] = dpEntry{totalPacks: 0, lastPack: 0, valid: true}

	for t := 1; t <= dpSize; t++ {
		best := dpEntry{valid: false}
		for _, p := range reduced {
			if p > t {
				continue
			}
			prev := dp[t-p]
			if !prev.valid {
				continue
			}
			if !best.valid || prev.totalPacks+1 < best.totalPacks {
				best = dpEntry{totalPacks: prev.totalPacks + 1, lastPack: p, valid: true}
			}
		}
		dp[t] = best
	}

	// Step 5: Find the optimal combination
	// Try using q copies of the largest pack, then fill the remainder from the DP table
	// We want: (1) minimize total items shipped, (2) break ties by fewest packs
	L := reduced[0] // largest reduced pack size
	qGreedy := rTarget / L

	// If rTarget is large, we must use at least qStart big packs
	// so the remainder fits within the DP table
	qStart := 0
	if rTarget > dpSize {
		qStart = (rTarget - dpSize + L - 1) / L
	}

	type solution struct {
		totalItems int
		numPacks   int
		q          int // how many largest packs
		dpIdx      int // remainder handled by DP
	}
	best := solution{totalItems: -1}

	for q := qGreedy; q >= qStart; q-- {
		need := rTarget - q*L
		if need < 0 {
			continue
		}
		// Look for the smallest achievable amount >= need in the DP table
		upper := min(need+L-1, dpSize)
		for t := need; t <= upper; t++ {
			if dp[t].valid {
				total := q*L + t
				packs := q + dp[t].totalPacks
				if best.totalItems < 0 ||
					total < best.totalItems ||
					(total == best.totalItems && packs < best.numPacks) {
					best = solution{totalItems: total, numPacks: packs, q: q, dpIdx: t}
				}
				break // first valid t gives minimum overshipping for this q
			}
		}
		if best.totalItems == rTarget {
			break // exact match found, no need to keep searching
		}
	}

	if best.totalItems < 0 {
		return nil, &InternalError{Err: fmt.Errorf("no valid packing found for %d items", items)}
	}

	// Step 6: Reconstruct the answer
	// Convert reduced pack sizes back to real sizes (multiply by GCD)
	result := make(PacksCount)
	if best.q > 0 {
		result[reduced[0]*g] = best.q
	}

	// Backtrack through the DP table to recover which smaller packs were used
	rem := best.dpIdx
	for rem > 0 {
		p := dp[rem].lastPack
		if p <= 0 {
			return nil, &InternalError{Err: fmt.Errorf("invalid backtrack at remainder %d", rem)}
		}
		result[p*g]++
		rem -= p
	}

	return result, nil
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
