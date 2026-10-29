package changes

import (
	"slices"
	"testing"
)

// summary lists changes as "M path" lines, sorted.
func summary(list []Change) []string {
	var out []string
	for _, ch := range list {
		s := string(ch.Status) + " " + ch.Path
		if ch.OldPath != "" {
			s += " <- " + ch.OldPath
		}
		if ch.Note != "" {
			s += " (" + ch.Note + ")"
		}
		out = append(out, s)
	}
	slices.Sort(out)
	return out
}

func TestParseNameStatus(t *testing.T) {
	out := "M\x00a.go\x00A\x00new.go\x00D\x00gone.go\x00R087\x00old.go\x00moved.go\x00C100\x00a.go\x00copy.go\x00T\x00link\x00"
	list, err := parseNameStatus([]byte(out))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A copy.go", "A new.go", "D gone.go", "M a.go", "M link", "R moved.go <- old.go"}
	if got := summary(list); !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	if _, err := parseNameStatus([]byte("R100\x00old.go\x00")); err == nil {
		t.Error("a rename without its new path should fail")
	}
}

func TestHash(t *testing.T) {
	a := Change{Path: "a.go", Status: StatusModified, Old: "x\n", New: "y\n"}
	same, edited := a, a
	edited.New = "z\n"
	if a.Hash() != same.Hash() || a.Hash() == edited.Hash() {
		t.Fatal("a change's hash should follow its texts")
	}
	// The texts' boundary counts: "ab" → "c" is not "a" → "bc".
	moved := Change{Path: "a.go", Status: StatusModified, Old: "x", New: "\ny\n"}
	if moved.Hash() == (Change{Path: "a.go", Status: StatusModified, Old: "x\n", New: "y\n"}).Hash() {
		t.Fatal("changes with the same bytes split differently should differ")
	}
}
