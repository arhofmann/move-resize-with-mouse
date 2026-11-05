package main

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"
)

type dragKind int

const (
	dragMove dragKind = iota
	dragResize
)

// Mouse buttons as bits.
const (
	buttonLeft uint32 = 1 << iota
	buttonRight
	buttonMiddle
)

type dragState struct {
	active  bool
	kind    dragKind
	button  uint32
	hwnd    uintptr
	mover   *mover
	origin  point // cursor position when the drag started
	last    point
	start   rect // window rectangle when the drag started
	restore bool // still maximized; un-maximize once the cursor really moves
	// Which edges follow the cursor while resizing: -1 left/top, +1 right/bottom, 0 neither.
	edgeX, edgeY int32
	minW, minH   int32
}

var (
	drag dragState // hook thread only
	// Buttons whose press was hidden from other applications (hook thread
	// only). Their release is hidden too, and no other release is.
	hiddenPresses     uint32
	lastDraggedApp    atomic.Value // string; set by the hook thread, read by the tray menu
	mouseHookProc     = syscall.NewCallback(lowLevelMouseProc)
	desktopSwitchProc = syscall.NewCallback(onDesktopSwitch)
	ownProcessID      = uint32(os.Getpid())
)

// Shell surfaces that must never be dragged around.
var ignoredClasses = map[string]bool{
	"Progman":                             true, // desktop
	"WorkerW":                             true, // desktop
	"Shell_TrayWnd":                       true, // taskbar
	"Shell_SecondaryTrayWnd":              true,
	"Windows.UI.Core.CoreWindow":          true, // Start menu, search, flyouts
	"NotifyIconOverflowWindow":            true,
	"TopLevelWindowForOverflowXamlIsland": true,
	"XamlExplorerHostIslandWindow":        true,
	"Shell_InputSwitchTopLevelWindow":     true,
	"MultitaskingViewFrame":               true,
	"TaskListThumbnailWnd":                true,
	"#32768":                              true, // popup menus
}

// startMouseHook installs the mouse hook on a thread that does nothing but
// wait for input. A low-level hook runs on the installing thread's message
// loop, so slow work there (tray updates talk to Explorer, settings go to
// disk) would stall the mouse system-wide, and Windows silently removes hooks
// that keep timing out. The hook stays installed until the process exits.
func startMouseHook() error {
	result := make(chan error)
	go func() {
		runtime.LockOSThread()
		h, _, err := procSetWindowsHookExW.Call(whMouseLL, mouseHookProc, moduleHandle(), 0)
		if h == 0 {
			result <- fmt.Errorf("SetWindowsHookEx: %w", err)
			return
		}
		procSetWinEventHook.Call(eventSystemDesktopSwitch, eventSystemDesktopSwitch, 0,
			desktopSwitchProc, 0, 0, wineventOutOfContext)
		result <- nil
		runMessageLoop()
	}()
	return <-result
}

func lowLevelMouseProc(nCode, wParam, lParam uintptr) uintptr {
	if int32(nCode) == hcAction {
		info := (*msllHookStruct)(unsafe.Pointer(lParam))
		if handleMouse(uint32(wParam), info.Pt) {
			return 1 // swallow the event
		}
	}
	r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, lParam)
	return r
}

// onDesktopSwitch ends a drag in progress: the lock screen, UAC prompts and
// Ctrl+Alt+Del run on another desktop, so the button-up never reaches the hook.
func onDesktopSwitch(hook, event, hwnd, object, child, thread, time uintptr) uintptr {
	if drag.active {
		endDrag()
	}
	return 0
}

// handleMouse runs for every mouse event system-wide and must return quickly.
// It reports whether the event should be hidden from other applications.
func handleMouse(m uint32, pt point) bool {
	if m == wmMouseMove {
		if drag.active {
			drag.update(pt)
		}
		return false
	}
	button, down := mouseButton(m)
	if button == 0 {
		return false
	}
	if !down {
		if drag.active && button == drag.button {
			endDrag()
		}
		if hiddenPresses&button != 0 {
			hiddenPresses &^= button
			return true
		}
		return false
	}

	hiddenPresses &^= button // a new press means the previous release got lost
	if drag.active {
		// Pressing another button ends the drag and is hidden along with it.
		// Pressing the drag's own button again means its release got lost
		// (e.g. to another program's hook); start over below.
		again := button == drag.button
		endDrag()
		if !again {
			hiddenPresses |= button
			return true
		}
	}
	if beginDrag(button, pt) {
		hiddenPresses |= button
		return true
	}
	return false
}

// mouseButton tells which button a message is about and whether it was pressed.
func mouseButton(m uint32) (button uint32, down bool) {
	switch m {
	case wmLButtonDown, wmLButtonUp:
		button = buttonLeft
	case wmRButtonDown, wmRButtonUp:
		button = buttonRight
	case wmMButtonDown, wmMButtonUp:
		button = buttonMiddle
	}
	return button, m == wmLButtonDown || m == wmRButtonDown || m == wmMButtonDown
}

func currentModifiers() uint32 {
	var m uint32
	if keyDown(vkMenu) {
		m |= modAlt
	}
	if keyDown(vkControl) {
		m |= modCtrl
	}
	if keyDown(vkShift) {
		m |= modShift
	}
	if keyDown(vkLWin) || keyDown(vkRWin) {
		m |= modWin
	}
	return m
}

func beginDrag(button uint32, pt point) bool {
	c := hookConfig.Load()
	if !c.Enabled || currentModifiers() != c.mods {
		return false
	}
	var kind dragKind
	switch {
	case button == buttonLeft:
		kind = dragMove
	case button == buttonRight && !c.MiddleClickResize, button == buttonMiddle && c.MiddleClickResize:
		kind = dragResize
	default:
		return false
	}

	hwnd := windowFromPoint(pt)
	if hwnd != 0 {
		hwnd = rootWindow(hwnd)
	}
	if hwnd == 0 || ignoredClasses[className(hwnd)] {
		return false
	}
	if _, pid := windowProcessID(hwnd); pid == ownProcessID {
		return false
	}
	app := processName(hwnd)
	if c.isExcluded(app) || isHungAppWindow(hwnd) {
		return false
	}
	style := windowStyle(hwnd)
	if kind == dragResize && style&wsThickFrame == 0 {
		return false // not resizable
	}
	r, ok := windowRect(hwnd)
	if !ok {
		return false
	}
	zoomed := isZoomed(hwnd)
	if !zoomed && isFullscreen(hwnd, r, style) {
		return false // leave games and fullscreen video alone
	}
	if zoomed && kind == dragMove {
		// Un-maximized once the cursor moves; the drag then continues from a normal window.
		r = restoredRectUnderCursor(hwnd, r, pt)
	}

	if c.mods&(modAlt|modWin) != 0 {
		sendMenuMaskKey()
	}

	drag = dragState{
		active:  true,
		kind:    kind,
		button:  button,
		hwnd:    hwnd,
		mover:   newMover(hwnd),
		origin:  pt,
		last:    pt,
		start:   r,
		restore: zoomed,
		minW:    systemMetric(smCxMinTrack),
		minH:    systemMetric(smCyMinTrack),
	}
	if kind == dragResize {
		drag.edgeX = section(pt.X-r.Left, r.width())
		drag.edgeY = section(pt.Y-r.Top, r.height())
		if drag.edgeX == 0 && drag.edgeY == 0 {
			drag.edgeX, drag.edgeY = 1, 1
		}
	}
	lastDraggedApp.Store(app)

	if c.BringToFront {
		drag.mover.submit(windowOp{hwnd: hwnd, activate: true})
	}
	return true
}

func endDrag() {
	drag.active = false
	drag.mover.finish()
	drag.mover = nil
}

// section splits a window into thirds: grabbing the outer third of an axis
// moves that edge, the middle third leaves the axis alone.
func section(offset, size int32) int32 {
	switch {
	case offset*3 < size:
		return -1
	case offset*3 > size*2:
		return 1
	}
	return 0
}

func (d *dragState) update(pt point) {
	if pt == d.last {
		return
	}
	d.last = pt
	dx, dy := pt.X-d.origin.X, pt.Y-d.origin.Y
	op := windowOp{hwnd: d.hwnd, setRect: true}
	if d.restore {
		// A plain click (or a twitch) must not un-maximize the window.
		if max(dx, -dx) <= systemMetric(smCxDrag) && max(dy, -dy) <= systemMetric(smCyDrag) {
			return
		}
		d.restore = false
		op.restore, op.resize = true, true
	}
	r := d.start

	if d.kind == dragMove {
		r = rect{r.Left + dx, r.Top + dy, r.Right + dx, r.Bottom + dy}
	} else {
		op.resize = true
		op.keepRight, op.keepBottom = d.edgeX < 0, d.edgeY < 0
		switch d.edgeX {
		case -1:
			r.Left = min(r.Left+dx, r.Right-d.minW)
		case 1:
			r.Right = max(r.Right+dx, r.Left+d.minW)
		}
		switch d.edgeY {
		case -1:
			r.Top = min(r.Top+dy, r.Bottom-d.minH)
		case 1:
			r.Bottom = max(r.Bottom+dy, r.Top+d.minH)
		}
	}
	op.rect = r
	d.mover.submit(op)
}

// restoredRectUnderCursor places the window's normal-size rectangle so the
// cursor keeps the same relative position it had in the maximized window.
func restoredRectUnderCursor(hwnd uintptr, maxRect rect, pt point) rect {
	n := normalRect(hwnd)
	w, h := n.width(), n.height()
	if w <= 0 || h <= 0 || maxRect.width() <= 0 || maxRect.height() <= 0 {
		return maxRect
	}
	left := pt.X - int32(int64(pt.X-maxRect.Left)*int64(w)/int64(maxRect.width()))
	top := pt.Y - int32(int64(pt.Y-maxRect.Top)*int64(h)/int64(maxRect.height()))
	return rect{left, top, left + w, top + h}
}

func isFullscreen(hwnd uintptr, r rect, style uintptr) bool {
	if style&wsCaption == wsCaption {
		return false
	}
	m := monitorRect(hwnd)
	return r.Left <= m.Left && r.Top <= m.Top && r.Right >= m.Right && r.Bottom >= m.Bottom
}
