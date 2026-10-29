package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

// snapshot saves a picture of the window once the files have loaded to
// the path OSPREY_SNAPSHOT names, and quits, for checking the interface
// without looking at the screen. OSPREY_SNAPSHOT_SETUP=file,page,layout,flat,history
// picks what it shows first, as "src/app.go,0,1,0,1". It fails, with the
// picture saved still, when the files don't load within a minute, or the
// file to show isn't among them.
func (a *app) snapshot() {
	out := os.Getenv("OSPREY_SNAPSHOT")
	if out == "" {
		return
	}
	go func() {
		start := time.Now()
		err := a.awaitLoad(time.Minute)
		if err == nil {
			fmt.Fprintf(os.Stderr, "snapshot: loaded in %.1fs\n", time.Since(start).Seconds())
			err = a.setUpSnapshot(os.Getenv("OSPREY_SNAPSHOT_SETUP"))
		}
		time.Sleep(1500 * time.Millisecond) // for the window to draw it
		png, saveErr := a.win.CapturePage()
		if saveErr == nil {
			saveErr = os.WriteFile(out, png, 0o644)
		}
		if err := errors.Join(err, saveErr); err != nil {
			fail(fmt.Errorf("snapshot: %w", err))
		}
		mygo.App.Quit()
	}()
}

// awaitLoad waits for the first load of the files to show, and returns its
// error, or one when it doesn't show within d.
func (a *app) awaitLoad(d time.Duration) error {
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(100 * time.Millisecond) {
		loaded := make(chan error, 1) // closed with nothing sent while loading
		a.win.Update(func() {
			if a.loaded || a.src == nil { // the welcome loads nothing
				loaded <- a.loadErr
			}
			close(loaded)
		})
		if err, ok := <-loaded; ok {
			return err
		}
	}
	return fmt.Errorf("the files didn't load within %v", d)
}

// setUpSnapshot shows what setup picks: file,page,layout,flat,history.
func (a *app) setUpSnapshot(setup string) error {
	var set [5]string
	copy(set[:], strings.Split(setup, ","))
	done := make(chan error, 1)
	a.win.Update(func() {
		if set[0] != "" {
			i := slices.IndexFunc(a.files, func(f *file) bool { return f.Path == set[0] })
			if i < 0 {
				done <- fmt.Errorf("%s is not among the files loaded", set[0])
				return
			}
			a.show(a.files[i])
		}
		a.page, _ = strconv.Atoi(set[1])   // pageDiff or pageFile
		a.layout, _ = strconv.Atoi(set[2]) // layoutSplit or layoutUnified
		a.flat = set[3] == "1"
		if set[4] == "1" {
			a.showHistory()
		}
		done <- nil
	})
	return <-done
}
