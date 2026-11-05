// Move Resize with Mouse: hold a modifier key (Alt by default) and drag
// anywhere inside a window to move it with the left mouse button or resize it
// with the right one. Lives in the notification area; right-click the icon for
// settings, left-click it to toggle.
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const appName = "Move Resize with Mouse"

// version is stamped in by release builds (build.ps1 -Version).
var version = "dev"

var instanceMutex uintptr

func init() {
	// The tray window must be serviced by the thread that created it: this one.
	runtime.LockOSThread()
}

func main() {
	if !acquireSingleInstance() {
		messageBox(ui.alreadyRunning, mbIconInformation)
		return
	}
	enableDPIAwareness()

	var firstRun bool
	cfg, firstRun = loadConfig()

	if err := createMainWindow(); err != nil {
		fatal(err)
	}
	updateTrayIcon(nimAdd)
	if err := startMouseHook(); err != nil {
		removeTrayIcon()
		fatal(err)
	}
	if firstRun {
		showBalloon(ui.firstRunTitle, fmt.Sprintf(ui.firstRunText, cfg.modifierLabel()))
	}

	runMessageLoop()
}

func acquireSingleInstance() bool {
	h, _, err := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(utf16Ptr(`Local\MoveResizeWithMouse`))))
	if h == 0 {
		return true
	}
	instanceMutex = h
	return err != syscall.Errno(errorAlreadyExists)
}

func runMessageLoop() {
	var m msg
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func fatal(err error) {
	messageBox(ui.startFailed+err.Error(), mbIconError)
	os.Exit(1)
}
