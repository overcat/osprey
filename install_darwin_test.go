package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

// The script that asks for an administrator's password compiles; running it
// would ask.
func TestFallbackScript(t *testing.T) {
	var args []string
	for _, line := range fallbackScript {
		args = append(args, "-e", line)
	}
	args = append(args, "-o", filepath.Join(t.TempDir(), "fallback.scpt"))
	if out, err := exec.Command("osacompile", args...).CombinedOutput(); err != nil {
		t.Fatalf("osacompile: %v\n%s", err, out)
	}
}
