package main

import (
	"runtime"
	"strings"
)

// The keys of Osprey's commands, as mygo's accelerators name them: the menu
// binds them, and tooltips and hints show them as each platform writes
// keys. Only the review's own have keys, with those everyone knows: the
// rest is a click away, in the toolbar, the sidebar and the menus.
const (
	keyReviewed       = "CmdOrCtrl+Enter"
	keyNextChange     = "Alt+Down"
	keyPreviousChange = "Alt+Up"
	keyNextFile       = "CmdOrCtrl+]"
	keyPreviousFile   = "CmdOrCtrl+["
	keyFind           = "CmdOrCtrl+P"
	keyFindInFile     = "CmdOrCtrl+F"
	keyRefresh        = "CmdOrCtrl+R"
	keyOpen           = "CmdOrCtrl+O"
	keyShortcuts      = "CmdOrCtrl+/"
)

// keySidebar shows or hides the sidebar: macOS's own key, and elsewhere
// Ctrl+B, as code editors there have it. Cmd is the Windows or Super key
// there, whose combinations the system keeps for itself.
var keySidebar = onMac("Ctrl+Cmd+S", "Ctrl+B")

func onMac(mac, other string) string {
	if runtime.GOOS == "darwin" {
		return mac
	}
	return other
}

// A shortcut is what keys do: one key, or a pair, as next and previous.
type shortcut struct {
	label string
	keys  []string
}

// shortcuts are the keys the shortcuts panel lists, by what they are for,
// the review's own first. A test checks that they are the menu's.
var shortcuts = []struct {
	group string
	keys  []shortcut
}{
	{"Review", []shortcut{
		{"Mark reviewed, go to the next", []string{keyReviewed}},
		{"Next / previous change", []string{keyNextChange, keyPreviousChange}},
		{"Next / previous file", []string{keyNextFile, keyPreviousFile}},
	}},
	{"Find", []shortcut{
		{"Find a file", []string{keyFind}},
		{"Find in the file", []string{keyFindInFile}},
	}},
	{"Window", []shortcut{
		{"Show or hide the sidebar", []string{keySidebar}},
		{"Refresh", []string{keyRefresh}},
		{"Open a repository", []string{keyOpen}},
		{"These shortcuts", []string{keyShortcuts}},
	}},
}

// withKey is a tooltip: what a control does, and its key, if it has one.
func withKey(label, key string) string {
	if key == "" {
		return label
	}
	return label + " (" + keyLabel(key) + ")"
}

// keyLabel writes a key as the platform does: ⇧⌘F on macOS, and
// Ctrl+Shift+F elsewhere.
func keyLabel(key string) string { return keyLabelOn(runtime.GOOS, key) }

func keyLabelOn(goos, key string) string {
	parts := strings.Split(key, "+")
	name := parts[len(parts)-1]
	var ctrl, alt, shift, cmd bool
	for _, mod := range parts[:len(parts)-1] {
		switch mod {
		case "CmdOrCtrl":
			cmd = goos == "darwin"
			ctrl = !cmd
		case "Cmd":
			cmd = true
		case "Ctrl":
			ctrl = true
		case "Alt":
			alt = true
		case "Shift":
			shift = true
		}
	}
	if goos == "darwin" {
		// In the order of macOS's menus: ⌃⌥⇧⌘.
		var b strings.Builder
		for _, m := range []struct {
			on    bool
			glyph string
		}{{ctrl, "⌃"}, {alt, "⌥"}, {shift, "⇧"}, {cmd, "⌘"}} {
			if m.on {
				b.WriteString(m.glyph)
			}
		}
		if glyph, ok := macKeys[name]; ok {
			name = glyph
		}
		return b.String() + name
	}
	var mods []string
	if ctrl {
		mods = append(mods, "Ctrl")
	}
	if cmd {
		super := "Super"
		if goos == "windows" {
			super = "Win"
		}
		mods = append(mods, super)
	}
	if alt {
		mods = append(mods, "Alt")
	}
	if shift {
		mods = append(mods, "Shift")
	}
	return strings.Join(append(mods, name), "+")
}

// macKeys are the glyphs macOS writes keys of names with.
var macKeys = map[string]string{"Enter": "↩", "Up": "↑", "Down": "↓", "Left": "←", "Right": "→"}
