package services

const (
	MaxItems     = 1_000_000_000 // MaxItems caps the items input to prevent expensive calculations
	MaxPackCount = 20            // MaxPackCount is the maximum number of distinct pack sizes allowed
	MaxPackSize  = 1_000_000     // MaxPackSize is the maximum value for a single pack size
)

// PacksCount maps pack size to quantity needed
type PacksCount = map[int]int

// InternalError marks errors caused by server-side failures
// (IO, bugs) rather than invalid user input
type InternalError struct {
	Err error
}

func (e *InternalError) Error() string { return e.Err.Error() }
func (e *InternalError) Unwrap() error { return e.Err }
