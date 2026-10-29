package review

import (
	"os"
	"path/filepath"
	"testing"
)

// isolate keeps the stores a test saves in a directory of its own.
func isolate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("AppData", filepath.Join(home, "AppData"))
}

func TestStore(t *testing.T) {
	isolate(t)
	s := Load("git\x00/somewhere")
	s.Set("a.go", "h1", true)
	if !s.Reviewed("a.go", "h1") || s.Reviewed("a.go", "h2") || s.Reviewed("b.go", "h1") {
		t.Fatal("a change reviewed should be, and only that change")
	}
	for i := range kept + 2 {
		s.Set("a.go", string(rune('a'+i)), true)
	}
	if len(s.marks["a.go"]) != kept || s.Reviewed("a.go", "h1") {
		t.Fatalf("kept %d reviews of a path", len(s.marks["a.go"]))
	}
	s.Set("a.go", "j", false)
	if s.Reviewed("a.go", "j") {
		t.Fatal("couldn't unmark a change")
	}
	s.SetDone("abc", true)

	again := Load("git\x00/somewhere")
	if !again.Done("abc") || len(again.marks["a.go"]) != kept-1 || !again.Reviewed("a.go", "i") {
		t.Fatal("the reviews weren't saved")
	}
	again.SetDone("abc", false)
	if Load("git\x00/somewhere").Done("abc") {
		t.Fatal("a commit unchecked stayed done")
	}
	if Load("git\x00/elsewhere").Reviewed("a.go", "i") {
		t.Fatal("another repository's reviews should be its own")
	}
}

func TestNewKeepsNothing(t *testing.T) {
	isolate(t)
	s := New()
	s.Set("a.go", "h1", true)
	s.SetDone("abc", true)
	if !s.Reviewed("a.go", "h1") || !s.Done("abc") {
		t.Fatal("a store in memory should keep its reviews while it lasts")
	}
	config, _ := os.UserConfigDir()
	if _, err := os.Stat(filepath.Join(config, "Osprey")); err == nil {
		t.Fatal("a store in memory wrote to disk")
	}
}
