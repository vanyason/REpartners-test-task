# Original Task

Build a **pack size calculator** — given a number of ordered items, determine the optimal combination of packs to ship.

Pack Sizes (default, configurable)
- 250 Items
- 500 Items
- 1000 Items
- 2000 Items
- 5000 Items

Business Rules (in priority order)
1. **Whole packs only** — packs cannot be broken open.
2. **Minimize total items sent** — send the least amount of items that fulfils the order (takes precedence over rule 3).
3. **Minimize number of packs** — among solutions with the same total items, prefer fewer packs.

Example Test Cases
| Items Ordered | Correct Packs                  | Why Others Are Wrong                        |
|---------------|--------------------------------|---------------------------------------------|
| 1             | 1 x 250                        | 1 x 500 = more items than necessary         |
| 250           | 1 x 250                        | 1 x 500 = more items than necessary         |
| 251           | 1 x 500                        | 2 x 250 = more packs than necessary         |
| 501           | 1 x 500 + 1 x 250              | 1 x 1000 = more items; 3 x 250 = more packs |
| 12001         | 2 x 5000 + 1 x 2000 + 1 x 250  | 3 x 5000 = more items than necessary        |

Technical Requirements
- **Backend**: Golang HTTP API
- **Frontend**: UI to interact with the API
- **Tests**: Unit tests for the packing algorithm
- **Flexibility**: Pack sizes must be configurable (add/remove/change) without code changes

# Algorithm Design (Bounded DP + Greedy)

At first glance, greedy sounds reasonable. However, it will break at `items=6`, `packs = [5,3]`

Problem is a modified [Coin Change](https://leetcode.com/problems/coin-change/) with two twists:
- Target is >= amount (can overshoot), not exact
- Optimize min items first, then min packs (two-level)
- Amount can be up to billions → standard O(amount) DP is too large

Solution: LeetCode 322 on a small remainder window + greedy for the bulk.
See also: [NeetCode explanation](https://neetcode.io/solutions/coin-change), [Go implementation](https://reintech.io/blog/coin-change-problem-in-go)

See [algo file](../pkg/services/calculator.go) for details

# Edge cases to cover

- UI (2 sessions) - concurrent pack changes
- algorithm should handle cases fast enough such as `pack = 1` and `items ordered = billion`
- handle 0 packs / duplicates / packs <= 0 / items <= 0 / etc...
- greedy approach will may not work. Example:
    - items: 6; packs: 5,3. Greedy would return 5:1, 3:1 - answer is 3:2
- Add some limits to the algo
- Graceful shutdown

# Design thoughts

- separate ui for **calculate** and **pack management**
- If packs changed concurrently during UI - show simple warning message
- packs should survive restart - save in file (RWmutex) (enough for test task)

# API

- GET `/api/v1/packs `
    - ← `{ "packs": [250, 500, 1000, 2000, 5000] }`
- POST `/api/v1/packs` 
    - → `{ "packs": [250, 500, 1000, 2000, 5000] }`
    - ← `{ "packs": [250, 500, 1000, 2000, 5000] }`
- POST `/api/v1/calculate`
    - → `{ "items": 501 }`
    - ← `{ "packs": {"500": 1, "250": 1}, "pack_sizes_used": [250, 500, 1000, 2000, 5000] }` - needed for UI to catch changes
- Error responses: 
    - `400 with { "error": "message" }`