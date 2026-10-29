package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

// snapshot saves a picture of the window once the files have loaded to
// the path OSPREY_SNAPSHOT names, and quits, for checking the interface
// without looking at the screen. OSPREY_SNAPSHOT_SETUP=file,page,layout,flat,history
// picks what it shows first, as "src/app.go,0,1,0,1".
func (a *app) snapshot() {
	out := os.Getenv("OSPREY_SNAPSHOT")
	if out == "" {
		return
	}
	go func() {
		time.Sleep(1500 * time.Millisecond)
		a.win.Update(func() {
			var set [5]string
			copy(set[:], strings.Split(os.Getenv("OSPREY_SNAPSHOT_SETUP"), ","))
			if set[0] != "" {
				a.selected = set[0]
				a.show(a.current())
			}
			a.page, _ = strconv.Atoi(set[1])   // pageDiff or pageFile
			a.layout, _ = strconv.Atoi(set[2]) // layoutSplit or layoutUnified
			a.flat = set[3] == "1"
			if set[4] == "1" {
				a.showHistory()
			}
		})
		time.Sleep(1500 * time.Millisecond)
		png, err := a.win.CapturePage()
		if err == nil {
			err = os.WriteFile(out, png, 0o644)
		}
		if err != nil {
			os.Stderr.WriteString("snapshot: " + err.Error() + "\n")
		}
		mygo.App.Quit()
	}()
}
