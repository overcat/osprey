package changes

import "testing"

// Osprey runs git as it loads and watches a repository: a console window
// for each would flash on the screen.
func TestCommandOpensNoWindow(t *testing.T) {
	a := command(t.TempDir(), "version").SysProcAttr
	if a == nil || a.CreationFlags&createNoWindow == 0 || !a.HideWindow {
		t.Fatalf("git would open a console window: %+v", a)
	}
}
