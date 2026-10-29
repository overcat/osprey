package main

import "testing"

func TestKeyLabel(t *testing.T) {
	for _, tc := range []struct{ key, mac, windows, linux string }{
		{"CmdOrCtrl+Shift+E", "⇧⌘E", "Ctrl+Shift+E", "Ctrl+Shift+E"},
		{keyNextChange, "⌥↓", "Alt+Down", "Alt+Down"},
		{keyReviewed, "⌘↩", "Ctrl+Enter", "Ctrl+Enter"},
		{"Alt+Z", "⌥Z", "Alt+Z", "Alt+Z"},
		{"Ctrl+Cmd+S", "⌃⌘S", "Ctrl+Win+S", "Ctrl+Super+S"},
		{"Ctrl+B", "⌃B", "Ctrl+B", "Ctrl+B"},
	} {
		for goos, want := range map[string]string{"darwin": tc.mac, "windows": tc.windows, "linux": tc.linux} {
			if got := keyLabelOn(goos, tc.key); got != want {
				t.Errorf("%s on %s: %q, want %q", tc.key, goos, got, want)
			}
		}
	}
}
