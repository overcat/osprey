package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain plays Osprey, for run to start, when OSP_TEST_APP is set: it
// prints its arguments and whether it was told of a terminal, and exits 3.
func TestMain(m *testing.M) {
	if os.Getenv("OSP_TEST_APP") != "" {
		fmt.Printf("args %q, terminal %q\n", os.Args[1:], os.Getenv("OSPREY_TERMINAL"))
		fmt.Fprintln(os.Stderr, "to stderr")
		os.Exit(3)
	}
	os.Exit(m.Run())
}

func TestRun(t *testing.T) {
	t.Setenv("OSP_TEST_APP", "1")
	t.Setenv("OSPREY_TERMINAL", "")
	for _, terminal := range []bool{true, false} {
		var stdout, stderr bytes.Buffer
		code := run(os.Args[0], []string{"main", "--wait"}, terminal, &stdout, &stderr)
		want := `args ["main" "--wait"], terminal ""`
		if terminal {
			want = `args ["main" "--wait"], terminal "1"`
		}
		if code != 3 || strings.TrimSpace(stdout.String()) != want || strings.TrimSpace(stderr.String()) != "to stderr" {
			t.Errorf("terminal %v: exit %d, stdout %q, stderr %q", terminal, code, stdout.String(), stderr.String())
		}
	}
}

func TestRunWithoutApp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(filepath.Join(t.TempDir(), "Osprey"), nil, false, &stdout, &stderr); code != 1 || !strings.HasPrefix(stderr.String(), "osprey: ") {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
}
