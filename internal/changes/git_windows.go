package changes

import "syscall"

// createNoWindow is CREATE_NO_WINDOW: a console program started with it
// runs without a console window.
const createNoWindow = 0x08000000

// windowless keeps git from opening a console window. Windows opens one
// for each console program a GUI program starts, as Osprey is, so one
// would flash on the screen each time Osprey reads the repository.
func windowless() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNoWindow, HideWindow: true}
}
