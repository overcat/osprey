package main

import (
	_ "embed"
	"os"
	"strings"

	"github.com/egoist/mygo"
)

//go:embed resources/icon.png
var iconPNG []byte

// setDockIcon gives the Dock Osprey's icon when the program runs outside
// its app bundle, as go run or a bare build does; a bundle's comes from
// its Info.plist.
func setDockIcon() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	if strings.Contains(exe, ".app/Contents/MacOS/") {
		return
	}
	mygo.App.Dock.SetIcon(iconPNG)
}
