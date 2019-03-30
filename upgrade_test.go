package main

import "testing"
import "github.com/Jguer/yay/v9/generic"
import "github.com/Jguer/yay/v9/conf"

func TestGetVersionDiff(t *testing.T) {
	conf.UseColor = true

	type versionPair struct {
		Old string
		New string
	}

	in := []versionPair{
		{"1-1", "1-1"},
		{"1-1", "2-1"},
		{"2-1", "1-1"},
		{"1-1", "1-2"},
		{"1-2", "1-1"},
		{"1.2.3-1", "1.2.4-1"},
		{"1.8rc1+6+g0f377f94-1", "1.8rc1+1+g7e949283-1"},
		{"1.8rc1+6+g0f377f94-1", "1.8rc2+1+g7e949283-1"},
		{"1.8rc2", "1.9rc1"},
		{"2.99.917+812+g75795523-1", "2.99.917+823+gd9bf46e4-1"},
		{"1.2.9-1", "1.2.10-1"},
		{"1.2.10-1", "1.2.9-1"},
		{"1.2-1", "1.2.1-1"},
		{"1.2.1-1", "1.2-1"},
		{"0.7-4", "0.7+4+gd8d8c67-1"},
		{"1.0.2_r0-1", "1.0.2_r0-2"},
		{"1.0.2_r0-1", "1.0.2_r1-1"},
		{"1.0.2_r0-1", "1.0.3_r0-1"},
	}

	out := []versionPair{
		{"1-1" + generic.Red(""), "1-1" + generic.Green("")},
		{generic.Red("1-1"), generic.Green("2-1")},
		{generic.Red("2-1"), generic.Green("1-1")},
		{"1-" + generic.Red("1"), "1-" + generic.Green("2")},
		{"1-" + generic.Red("2"), "1-" + generic.Green("1")},
		{"1.2." + generic.Red("3-1"), "1.2." + generic.Green("4-1")},
		{"1.8rc1+" + generic.Red("6+g0f377f94-1"), "1.8rc1+" + generic.Green("1+g7e949283-1")},
		{"1.8" + generic.Red("rc1+6+g0f377f94-1"), "1.8" + generic.Green("rc2+1+g7e949283-1")},
		{"1." + generic.Red("8rc2"), "1." + generic.Green("9rc1")},
		{"2.99.917+" + generic.Red("812+g75795523-1"), "2.99.917+" + generic.Green("823+gd9bf46e4-1")},
		{"1.2." + generic.Red("9-1"), "1.2." + generic.Green("10-1")},
		{"1.2." + generic.Red("10-1"), "1.2." + generic.Green("9-1")},
		{"1.2" + generic.Red("-1"), "1.2" + generic.Green(".1-1")},
		{"1.2" + generic.Red(".1-1"), "1.2" + generic.Green("-1")},
		{"0.7" + generic.Red("-4"), "0.7" + generic.Green("+4+gd8d8c67-1")},
		{"1.0.2_r0-" + generic.Red("1"), "1.0.2_r0-" + generic.Green("2")},
		{"1.0.2_" + generic.Red("r0-1"), "1.0.2_" + generic.Green("r1-1")},
		{"1.0." + generic.Red("2_r0-1"), "1.0." + generic.Green("3_r0-1")},
	}

	for i, pair := range in {
		o, n := getVersionDiff(pair.Old, pair.New)

		if o != out[i].Old || n != out[i].New {
			t.Errorf("Test %d failed for update: expected (%s => %s) got (%s => %s) %d %d %d %d", i+1, in[i].Old, in[i].New, o, n, len(in[i].Old), len(in[i].New), len(o), len(n))
		}
	}
}
