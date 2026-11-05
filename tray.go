package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	wmTrayIcon = wmApp + 1
	trayIconID = 1
)

// Tray menu command IDs.
const (
	cmdEnabled = iota + 1
	cmdBringToFront
	cmdMiddleResize
	cmdAutostart
	cmdExcludeLast
	cmdQuit
	cmdModifierBase = 100 // + index into modifiers
	cmdExcludedBase = 200 // + index into cfg.ExcludedApps
)

var (
	mainWindow        uintptr
	taskbarCreatedMsg uint32
	iconEnabled       uintptr
	iconDisabled      uintptr
	wndProcCallback   = syscall.NewCallback(wndProc)
)

// createMainWindow creates the hidden window that owns the tray icon.
func createMainWindow() error {
	hInst := moduleHandle()
	class := utf16Ptr("EasyMoveResizeWindow")
	wc := wndClassExW{LpfnWndProc: wndProcCallback, HInstance: hInst, LpszClassName: class}
	wc.CbSize = uint32(unsafe.Sizeof(wc))
	if r, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return fmt.Errorf("RegisterClassEx: %w", err)
	}
	hwnd, _, err := procCreateWindowExW.Call(wsExToolWindow, uintptr(unsafe.Pointer(class)),
		uintptr(unsafe.Pointer(utf16Ptr(appName))), wsPopup, 0, 0, 0, 0, 0, 0, hInst, 0)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx: %w", err)
	}
	mainWindow = hwnd

	// Explorer broadcasts this after restarting; the tray icon must then be re-added.
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(utf16Ptr("TaskbarCreated"))))
	taskbarCreatedMsg = uint32(r)
	procChangeWindowMessageFilterEx.Call(hwnd, uintptr(taskbarCreatedMsg), msgfltAllow, 0)

	size := int(systemMetric(smCxSmIcon))
	iconEnabled = makeIcon(size, 0x2F, 0x6F, 0xEB)
	iconDisabled = makeIcon(size, 0x8A, 0x8A, 0x8A)
	return nil
}

func wndProc(hwnd, message, wParam, lParam uintptr) uintptr {
	switch m := uint32(message); {
	case m == wmTrayIcon:
		switch uint32(lParam) {
		case wmLButtonUp:
			setEnabled(!cfg.Enabled)
		case wmRButtonUp:
			showTrayMenu()
		}
		return 0
	case m == taskbarCreatedMsg && m != 0:
		updateTrayIcon(nimAdd)
		return 0
	case m == wmDestroy:
		removeTrayIcon()
		procPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, message, wParam, lParam)
	return r
}

func trayData() notifyIconData {
	nid := notifyIconData{HWnd: mainWindow, UID: trayIconID}
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	return nid
}

// updateTrayIcon adds (nimAdd) or refreshes (nimModify) the tray icon.
func updateTrayIcon(action uintptr) {
	nid := trayData()
	nid.UFlags = nifMessage | nifIcon | nifTip
	nid.UCallbackMessage = wmTrayIcon
	nid.HIcon = iconEnabled
	tip := ui.tipDisabled
	if cfg.Enabled {
		resizeButton := ui.tipRightButton
		if cfg.MiddleClickResize {
			resizeButton = ui.tipMiddleButton
		}
		tip = fmt.Sprintf(ui.tipEnabled, cfg.modifierLabel(), resizeButton)
	} else {
		nid.HIcon = iconDisabled
	}
	copyUTF16(nid.SzTip[:], tip)
	procShellNotifyIconW.Call(action, uintptr(unsafe.Pointer(&nid)))
}

func removeTrayIcon() {
	nid := trayData()
	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func showBalloon(title, text string) {
	nid := trayData()
	nid.UFlags = nifInfo
	nid.DwInfoFlags = niifInfo
	copyUTF16(nid.SzInfoTitle[:], title)
	copyUTF16(nid.SzInfo[:], text)
	procShellNotifyIconW.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func setEnabled(on bool) {
	cfg.Enabled = on
	cfg.save()
	updateTrayIcon(nimModify)
}

func appendMenuItem(menu uintptr, id int, label string, checked, grayed bool) {
	flags := uintptr(mfString)
	if checked {
		flags |= mfChecked
	}
	if grayed {
		flags |= mfGrayed
	}
	procAppendMenuW.Call(menu, flags, uintptr(id), uintptr(unsafe.Pointer(utf16Ptr(label))))
}

func appendSubmenu(menu, sub uintptr, label string) {
	procAppendMenuW.Call(menu, mfPopup, sub, uintptr(unsafe.Pointer(utf16Ptr(label))))
}

func appendSeparator(menu uintptr) {
	procAppendMenuW.Call(menu, mfSeparator, 0, 0)
}

func showTrayMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	defer procDestroyMenu.Call(menu) // also destroys the submenus

	modMenu, _, _ := procCreatePopupMenu.Call()
	for i, m := range modifiers {
		appendMenuItem(modMenu, cmdModifierBase+i, m.label, cfg.mods&m.bit != 0, false)
	}

	exclMenu, _, _ := procCreatePopupMenu.Call()
	for i, app := range cfg.ExcludedApps {
		appendMenuItem(exclMenu, cmdExcludedBase+i, app, true, false)
	}
	if len(cfg.ExcludedApps) == 0 {
		appendMenuItem(exclMenu, 0, ui.menuExcludedNone, false, true)
	}
	lastApp, _ := lastDraggedApp.Load().(string)
	if lastApp != "" && !cfg.isExcluded(lastApp) {
		appendSeparator(exclMenu)
		appendMenuItem(exclMenu, cmdExcludeLast, fmt.Sprintf(ui.menuExclude, lastApp), false, false)
	}

	appendMenuItem(menu, 0, appName+" "+version, false, true)
	appendSeparator(menu)
	appendMenuItem(menu, cmdEnabled, ui.menuEnabled, cfg.Enabled, false)
	appendSubmenu(menu, modMenu, ui.menuModifier+cfg.modifierLabel())
	appendMenuItem(menu, cmdBringToFront, ui.menuBringToFront, cfg.BringToFront, false)
	appendMenuItem(menu, cmdMiddleResize, ui.menuMiddleResize, cfg.MiddleClickResize, false)
	appendSubmenu(menu, exclMenu, ui.menuExcluded)
	appendSeparator(menu)
	appendMenuItem(menu, cmdAutostart, ui.menuAutostart, autostartEnabled(), false)
	appendMenuItem(menu, cmdQuit, ui.menuQuit, false, false)

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	// Without this the menu would not close when clicking elsewhere.
	procSetForegroundWindow.Call(mainWindow)
	cmd, _, _ := procTrackPopupMenu.Call(menu, tpmReturnCmd|tpmRightButton|tpmNoNotify,
		uintptr(pt.X), uintptr(pt.Y), 0, mainWindow, 0)
	procPostMessageW.Call(mainWindow, wmNull, 0, 0)

	handleCommand(int(cmd), lastApp)
}

// handleCommand runs a tray menu command; lastApp is the program the menu
// offered to exclude.
func handleCommand(cmd int, lastApp string) {
	switch {
	case cmd == cmdEnabled:
		setEnabled(!cfg.Enabled)
	case cmd == cmdBringToFront:
		cfg.BringToFront = !cfg.BringToFront
		cfg.save()
	case cmd == cmdMiddleResize:
		cfg.MiddleClickResize = !cfg.MiddleClickResize
		cfg.save()
		updateTrayIcon(nimModify)
	case cmd == cmdAutostart:
		setAutostart(!autostartEnabled())
	case cmd == cmdExcludeLast:
		cfg.setExcluded(lastApp, true)
	case cmd == cmdQuit:
		procDestroyWindow.Call(mainWindow)
	case cmd >= cmdModifierBase && cmd < cmdModifierBase+len(modifiers):
		mods := cfg.mods ^ modifiers[cmd-cmdModifierBase].bit
		if mods != 0 { // at least one modifier is required
			cfg.mods = mods
			cfg.save()
			updateTrayIcon(nimModify)
		}
	case cmd >= cmdExcludedBase && cmd < cmdExcludedBase+len(cfg.ExcludedApps):
		cfg.setExcluded(cfg.ExcludedApps[cmd-cmdExcludedBase], false)
	}
}
