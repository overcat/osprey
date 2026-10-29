//go:build !windows

package changes

import "syscall"

// windowless is nil: only Windows opens a window for git.
func windowless() *syscall.SysProcAttr { return nil }
