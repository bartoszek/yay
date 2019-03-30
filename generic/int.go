package generic

import (
	"sync"
	"unicode"
)

// IntRange represents a range between two numbers
type IntRange struct {
	Min int
	Max int
}

// MakeIntRange creates an IntRange from two input values
func MakeIntRange(min, max int) IntRange {
	return IntRange{
		min,
		max,
	}
}

func (r IntRange) get(n int) bool {
	return n >= r.Min && n <= r.Max
}

// IntRanges , a slice of IntRanges
type IntRanges []IntRange

// Get checks the existence of a n IntRange
func (rs IntRanges) Get(n int) bool {
	for _, r := range rs {
		if r.get(n) {
			return true
		}
	}

	return false
}

// Min wannabe macro
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max wannabe macro
func Max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

// LessRunes rune slice comparison
func LessRunes(iRunes, jRunes []rune) bool {
	max := len(iRunes)
	if max > len(jRunes) {
		max = len(jRunes)
	}

	for idx := 0; idx < max; idx++ {
		ir := iRunes[idx]
		jr := jRunes[idx]

		lir := unicode.ToLower(ir)
		ljr := unicode.ToLower(jr)

		if lir != ljr {
			return lir < ljr
		}

		// the lowercase runes are the same, so compare the original
		if ir != jr {
			return ir < jr
		}
	}

	return len(iRunes) < len(jRunes)
}

// MultiError encapsulates various errors encountered
type MultiError struct {
	Errors []error
	mux    sync.Mutex
}

func (err *MultiError) Error() string {
	str := ""

	for _, e := range err.Errors {
		str += e.Error() + "\n"
	}

	return str[:len(str)-1]
}

// Add inserts a new error in the encapsulation
func (err *MultiError) Add(e error) {
	if e == nil {
		return
	}

	err.mux.Lock()
	err.Errors = append(err.Errors, e)
	err.mux.Unlock()
}

// Return handles returns of functions in case of error
func (err *MultiError) Return() error {
	if len(err.Errors) > 0 {
		return err
	}

	return nil
}
