package main

import (
	"strings"
	"testing"
)

func TestCompletion(t *testing.T) {
	for _, shell := range []string{"fish", "zsh", "bash"} {
		var script strings.Builder
		if err := completion(&script, shell); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"osprey", "osp", "committed", "wait", "for-each-ref"} {
			if !strings.Contains(script.String(), want) {
				t.Errorf("%s: no %q in\n%s", shell, want, script.String())
			}
		}
		// zsh's completes the arguments of a comparison alone.
		if shell != "zsh" && (!strings.Contains(script.String(), "install-cli") || !strings.Contains(script.String(), "uninstall-cli")) {
			t.Errorf("%s: no install-cli and uninstall-cli in\n%s", shell, script.String())
		}
	}
	// fish loads a command's completions from the file of its name.
	var fish strings.Builder
	completion(&fish, "fish", "osp")
	if strings.Contains(fish.String(), "complete -c osprey ") {
		t.Error("osp's fish script completes osprey too")
	}
	if err := completion(&fish, "tcsh"); err == nil {
		t.Error("tcsh has no script")
	}
}
