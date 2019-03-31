package generic

import (
	"fmt"
	"strings"

	"github.com/Jguer/yay/v9/conf"
)

// StringSet is a basic set implementation for strings.
// This is used a lot so it deserves its own type.
// Other types of sets are used throughout the code but do not have
// their own typedef.
// String sets and <type>sets should be used throughout the code when applicable,
// they are a lot more flexible than slices and provide easy lookup.
type StringSet map[string]struct{}

// MapStringSet represents a map of string sets
type MapStringSet map[string]StringSet

// Set places an empty Set in the input key
func (set StringSet) Set(v string) {
	set[v] = struct{}{}
}

// Get returns true if v exists in set
func (set StringSet) Get(v string) bool {
	_, exists := set[v]
	return exists
}

// Remove removes key v from StringSet
func (set StringSet) Remove(v string) {
	delete(set, v)
}

// ToSlice transforms StringSet to slice
func (set StringSet) ToSlice() []string {
	slice := make([]string, 0, len(set))

	for v := range set {
		slice = append(slice, v)
	}

	return slice
}

// Copy creates a clone of the StringSet
func (set StringSet) Copy() StringSet {
	newSet := make(StringSet)

	for str := range set {
		newSet.Set(str)
	}

	return newSet
}

// SliceToStringSet transforms an explicit slice to a StringSet
func SliceToStringSet(in []string) StringSet {
	set := make(StringSet)

	for _, v := range in {
		set.Set(v)
	}

	return set
}

// MakeStringSet transforms a slice to a StringSet
func MakeStringSet(in ...string) StringSet {
	return SliceToStringSet(in)
}

// Add inserts a string set from two input strings
func (mss MapStringSet) Add(n string, v string) {
	_, ok := mss[n]
	if !ok {
		mss[n] = make(StringSet)
	}
	mss[n].Set(v)
}

// StringSliceEqual compares two slices for equality
func StringSliceEqual(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// ContinueTask prompts if user wants to continue task.
//If NoConfirm is set the action will continue without user input.
func ContinueTask(s string, cont bool) bool {
	if conf.Current.NoConfirm {
		return cont
	}

	var response string
	var postFix string
	yes := "yes"
	no := "no"
	y := string([]rune(yes)[0])
	n := string([]rune(no)[0])

	if cont {
		postFix = fmt.Sprintf(" [%s/%s] ", strings.ToUpper(y), n)
	} else {
		postFix = fmt.Sprintf(" [%s/%s] ", y, strings.ToUpper(n))
	}

	fmt.Print(Bold(Green(Arrow)+" "+s), Bold(postFix))

	if _, err := fmt.Scanln(&response); err != nil {
		return cont
	}

	response = strings.ToLower(response)
	return response == yes || response == y
}
