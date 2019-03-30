package main

import "testing"
import "github.com/Jguer/yay/v9/generic"

func intRangesEqual(a, b generic.IntRanges) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for n := range a {
		r1 := a[n]
		r2 := b[n]

		if r1.Min != r2.Min || r1.Max != r2.Max {
			return false
		}
	}

	return true
}

func stringSetEqual(a, b generic.StringSet) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}

	for n := range a {
		if !b.Get(n) {
			return false
		}
	}

	return true
}
func TestParseNumberMenu(t *testing.T) {
	type result struct {
		Include      generic.IntRanges
		Exclude      generic.IntRanges
		OtherInclude generic.StringSet
		OtherExclude generic.StringSet
	}

	inputs := []string{
		"1 2 3 4 5",
		"1-10 5-15",
		"10-5 90-85",
		"1 ^2 ^10-5 99 ^40-38 ^123 60-62",
		"abort all none",
		"a-b ^a-b ^abort",
		"1\t2   3      4\t\t  \t 5",
		"1 2,3, 4,  5,6 ,7  ,8",
		"",
		"   \t   ",
		"A B C D E",
	}

	expected := []result{
		{generic.IntRanges{generic.MakeIntRange(1, 1), generic.MakeIntRange(2, 2), generic.MakeIntRange(3, 3), generic.MakeIntRange(4, 4), generic.MakeIntRange(5, 5)}, generic.IntRanges{}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{generic.MakeIntRange(1, 10), generic.MakeIntRange(5, 15)}, generic.IntRanges{}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{generic.MakeIntRange(5, 10), generic.MakeIntRange(85, 90)}, generic.IntRanges{}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{generic.MakeIntRange(1, 1), generic.MakeIntRange(99, 99), generic.MakeIntRange(60, 62)}, generic.IntRanges{generic.MakeIntRange(2, 2), generic.MakeIntRange(5, 10), generic.MakeIntRange(38, 40), generic.MakeIntRange(123, 123)}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{}, generic.IntRanges{}, generic.MakeStringSet("abort", "all", "none"), make(generic.StringSet)},
		{generic.IntRanges{}, generic.IntRanges{}, generic.MakeStringSet("a-b"), generic.MakeStringSet("abort", "a-b")},
		{generic.IntRanges{generic.MakeIntRange(1, 1), generic.MakeIntRange(2, 2), generic.MakeIntRange(3, 3), generic.MakeIntRange(4, 4), generic.MakeIntRange(5, 5)}, generic.IntRanges{}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{generic.MakeIntRange(1, 1), generic.MakeIntRange(2, 2), generic.MakeIntRange(3, 3), generic.MakeIntRange(4, 4), generic.MakeIntRange(5, 5), generic.MakeIntRange(6, 6), generic.MakeIntRange(7, 7), generic.MakeIntRange(8, 8)}, generic.IntRanges{}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{}, generic.IntRanges{}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{}, generic.IntRanges{}, make(generic.StringSet), make(generic.StringSet)},
		{generic.IntRanges{}, generic.IntRanges{}, generic.MakeStringSet("a", "b", "c", "d", "e"), make(generic.StringSet)},
	}

	for n, in := range inputs {
		res := expected[n]
		include, exclude, otherInclude, otherExclude := parseNumberMenu(in)

		if !intRangesEqual(include, res.Include) ||
			!intRangesEqual(exclude, res.Exclude) ||
			!stringSetEqual(otherInclude, res.OtherInclude) ||
			!stringSetEqual(otherExclude, res.OtherExclude) {

			t.Fatalf("Test %d Failed: Expected: include=%+v exclude=%+v otherInclude=%+v otherExclude=%+v got include=%+v excluive=%+v otherInclude=%+v otherExclude=%+v",
				n+1, res.Include, res.Exclude, res.OtherInclude, res.OtherExclude, include, exclude, otherInclude, otherExclude)
		}
	}
}
