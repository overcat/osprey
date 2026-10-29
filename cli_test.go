package main

import (
	"errors"
	"slices"
	"testing"
)

func TestParseArgs(t *testing.T) {
	o, err := parseArgs([]string{"main", "-C", "/repo", "--committed", "--", "--weird-branch"})
	if err != nil {
		t.Fatal(err)
	}
	if o.dir != "/repo" || !o.committed || o.wait || !slices.Equal(o.args, []string{"main", "--", "--weird-branch"}) {
		t.Fatalf("%+v", o)
	}
	if o, _ := parseArgs([]string{"--dir=/repo", "-w", "a", "b"}); o.dir != "/repo" || !o.wait || len(o.args) != 2 {
		t.Fatalf("%+v", o)
	}
	for args, want := range map[string]error{"--help": errHelp, "-v": errVersion} {
		if _, err := parseArgs([]string{args}); !errors.Is(err, want) {
			t.Errorf("%s: %v", args, err)
		}
	}
	for _, args := range [][]string{{"--nope"}, {"-C"}} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("%q should fail", args)
		}
	}
}
