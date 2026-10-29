package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// installCLI puts the folder of Osprey.exe on the user's PATH, and says
// so: osp.exe and osprey.com beside it, console programs that run
// Osprey.exe, are then the osp and osprey commands of new terminals.
// Windows makes no links without privileges, so there is nothing to put
// elsewhere, and dir must be "".
func installCLI(dir string) (string, error) {
	if dir != "" {
		return "", errors.New("on Windows, install-cli puts the folder of Osprey.exe on your PATH, and takes no directory")
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	folder := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(folder, "osp.exe")); err != nil {
		return "", fmt.Errorf("osp.exe isn't beside %s: install Osprey from its release", exe)
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()
	path, kind, err := key.GetStringValue("Path")
	switch {
	case errors.Is(err, registry.ErrNotExist):
		kind = registry.EXPAND_SZ
	case err != nil:
		return "", err
	}
	if pathHas(path, folder) {
		return onPath(folder), nil // there already
	}
	if path = strings.TrimSuffix(path, ";"); path != "" {
		path += ";"
	}
	path += folder
	// The same kind of value as before, so that the %VARIABLES% of an
	// expandable PATH stay as they were.
	if kind == registry.SZ {
		err = key.SetStringValue("Path", path)
	} else {
		err = key.SetExpandStringValue("Path", path)
	}
	if err != nil {
		return "", err
	}
	announceEnvironment()
	return onPath(folder), nil
}

func onPath(folder string) string {
	return folder + " is on your PATH: osprey and osp work in new terminals."
}

// cliMissing reports whether the folder of Osprey.exe, with osp.exe and
// osprey.com, isn't on the PATH yet: the user's, as the registry keeps it
// for new terminals, or this program's.
func cliMissing() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	folder := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(folder, "osp.exe")); err != nil {
		return false // nothing to put on the PATH
	}
	if pathHas(os.Getenv("PATH"), folder) {
		return false
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, "Environment", registry.QUERY_VALUE)
	if err != nil {
		return true
	}
	defer key.Close()
	path, _, _ := key.GetStringValue("Path")
	return !pathHas(path, folder)
}

// pathHas reports whether the folders of path, a PATH, hold folder.
func pathHas(path, folder string) bool {
	for _, p := range filepath.SplitList(path) {
		if expanded, err := registry.ExpandString(p); err == nil && strings.EqualFold(filepath.Clean(expanded), folder) {
			return true
		}
	}
	return false
}

// announceEnvironment tells the programs running, Explorer among them, that
// the user's environment changed, so that the terminals they start next
// have the new PATH.
func announceEnvironment() {
	const (
		hwndBroadcast   = 0xffff
		wmSettingChange = 0x001a
		smtoAbortIfHung = 0x0002
	)
	name, _ := windows.UTF16PtrFromString("Environment")
	var result uintptr
	windows.NewLazySystemDLL("user32.dll").NewProc("SendMessageTimeoutW").Call(
		hwndBroadcast, wmSettingChange, 0, uintptr(unsafe.Pointer(name)), smtoAbortIfHung, 5000, uintptr(unsafe.Pointer(&result)))
}
