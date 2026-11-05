package main

// Minimal Win32 bindings via syscall, so the app builds without any third-party
// modules. Struct layouts and register-passed POINTs assume a 64-bit target
// (amd64 or arm64).

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	modShell32  = syscall.NewLazyDLL("shell32.dll")
	modAdvapi32 = syscall.NewLazyDLL("advapi32.dll")

	procAppendMenuW                   = modUser32.NewProc("AppendMenuW")
	procAttachThreadInput             = modUser32.NewProc("AttachThreadInput")
	procBringWindowToTop              = modUser32.NewProc("BringWindowToTop")
	procCallNextHookEx                = modUser32.NewProc("CallNextHookEx")
	procChangeWindowMessageFilterEx   = modUser32.NewProc("ChangeWindowMessageFilterEx")
	procCreateIconFromResourceEx      = modUser32.NewProc("CreateIconFromResourceEx")
	procCreatePopupMenu               = modUser32.NewProc("CreatePopupMenu")
	procCreateWindowExW               = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW                = modUser32.NewProc("DefWindowProcW")
	procDestroyMenu                   = modUser32.NewProc("DestroyMenu")
	procDestroyWindow                 = modUser32.NewProc("DestroyWindow")
	procDispatchMessageW              = modUser32.NewProc("DispatchMessageW")
	procGetAncestor                   = modUser32.NewProc("GetAncestor")
	procGetAsyncKeyState              = modUser32.NewProc("GetAsyncKeyState")
	procGetClassNameW                 = modUser32.NewProc("GetClassNameW")
	procGetCursorPos                  = modUser32.NewProc("GetCursorPos")
	procGetForegroundWindow           = modUser32.NewProc("GetForegroundWindow")
	procGetMessageW                   = modUser32.NewProc("GetMessageW")
	procGetMonitorInfoW               = modUser32.NewProc("GetMonitorInfoW")
	procGetSystemMetrics              = modUser32.NewProc("GetSystemMetrics")
	procGetWindowLongPtrW             = modUser32.NewProc("GetWindowLongPtrW")
	procGetWindowPlacement            = modUser32.NewProc("GetWindowPlacement")
	procGetWindowRect                 = modUser32.NewProc("GetWindowRect")
	procGetWindowThreadProcessId      = modUser32.NewProc("GetWindowThreadProcessId")
	procIsHungAppWindow               = modUser32.NewProc("IsHungAppWindow")
	procIsZoomed                      = modUser32.NewProc("IsZoomed")
	procMessageBoxW                   = modUser32.NewProc("MessageBoxW")
	procMonitorFromWindow             = modUser32.NewProc("MonitorFromWindow")
	procPostMessageW                  = modUser32.NewProc("PostMessageW")
	procPostQuitMessage               = modUser32.NewProc("PostQuitMessage")
	procRegisterClassExW              = modUser32.NewProc("RegisterClassExW")
	procRegisterWindowMessageW        = modUser32.NewProc("RegisterWindowMessageW")
	procSendInput                     = modUser32.NewProc("SendInput")
	procSetForegroundWindow           = modUser32.NewProc("SetForegroundWindow")
	procSetProcessDPIAware            = modUser32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwarenessContext = modUser32.NewProc("SetProcessDpiAwarenessContext")
	procSetWindowPos                  = modUser32.NewProc("SetWindowPos")
	procSetWindowsHookExW             = modUser32.NewProc("SetWindowsHookExW")
	procSetWinEventHook               = modUser32.NewProc("SetWinEventHook")
	procShowWindow                    = modUser32.NewProc("ShowWindow")
	procTrackPopupMenu                = modUser32.NewProc("TrackPopupMenu")
	procTranslateMessage              = modUser32.NewProc("TranslateMessage")
	procWindowFromPoint               = modUser32.NewProc("WindowFromPoint")

	procCloseHandle                = modKernel32.NewProc("CloseHandle")
	procCreateMutexW               = modKernel32.NewProc("CreateMutexW")
	procGetCurrentThreadId         = modKernel32.NewProc("GetCurrentThreadId")
	procGetModuleHandleW           = modKernel32.NewProc("GetModuleHandleW")
	procGetUserDefaultUILanguage   = modKernel32.NewProc("GetUserDefaultUILanguage")
	procOpenProcess                = modKernel32.NewProc("OpenProcess")
	procQueryFullProcessImageNameW = modKernel32.NewProc("QueryFullProcessImageNameW")

	procShellNotifyIconW = modShell32.NewProc("Shell_NotifyIconW")

	procRegDeleteKeyValueW = modAdvapi32.NewProc("RegDeleteKeyValueW")
	procRegGetValueW       = modAdvapi32.NewProc("RegGetValueW")
	procRegSetKeyValueW    = modAdvapi32.NewProc("RegSetKeyValueW")
)

const (
	whMouseLL = 14
	hcAction  = 0

	eventSystemDesktopSwitch = 0x0020
	wineventOutOfContext     = 0x0000

	wmNull        = 0x0000
	wmDestroy     = 0x0002
	wmMouseMove   = 0x0200
	wmLButtonDown = 0x0201
	wmLButtonUp   = 0x0202
	wmRButtonDown = 0x0204
	wmRButtonUp   = 0x0205
	wmMButtonDown = 0x0207
	wmMButtonUp   = 0x0208
	wmApp         = 0x8000

	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12
	vkLWin    = 0x5B
	vkRWin    = 0x5C
	// Unassigned virtual key, used to keep Alt/Win from opening menus on release.
	vkMenuMask = 0xE8

	gaRoot = 2

	gwlStyle = -16

	wsPopup        = 0x80000000
	wsCaption      = 0x00C00000
	wsThickFrame   = 0x00040000
	wsExToolWindow = 0x00000080

	swRestore = 9

	swpNoSize        = 0x0001
	swpNoZOrder      = 0x0004
	swpNoActivate    = 0x0010
	swpNoOwnerZOrder = 0x0200

	smCxMinTrack = 34
	smCyMinTrack = 35
	smCxSmIcon   = 49
	smCxDrag     = 68
	smCyDrag     = 69

	monitorDefaultToNearest = 2

	inputKeyboard  = 1
	keyEventFKeyUp = 0x0002

	nimAdd     = 0
	nimModify  = 1
	nimDelete  = 2
	nifMessage = 0x01
	nifIcon    = 0x02
	nifTip     = 0x04
	nifInfo    = 0x10
	niifInfo   = 0x01

	mfString    = 0x0000
	mfGrayed    = 0x0001
	mfChecked   = 0x0008
	mfPopup     = 0x0010
	mfSeparator = 0x0800

	tpmRightButton = 0x0002
	tpmNoNotify    = 0x0080
	tpmReturnCmd   = 0x0100

	mbIconError       = 0x10
	mbIconInformation = 0x40

	msgfltAllow = 1

	processQueryLimitedInformation = 0x1000

	rrfRtRegSz = 0x00000002
	regSz      = 1

	errorAlreadyExists = 183

	// Predefined handles are sign-extended 32-bit values.
	hkeyCurrentUser                      = ^uintptr(0x7FFFFFFE) // (HKEY)0x80000001
	dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)          // (DPI_AWARENESS_CONTEXT)-4
)

type point struct{ X, Y int32 }

type rect struct{ Left, Top, Right, Bottom int32 }

func (r rect) width() int32  { return r.Right - r.Left }
func (r rect) height() int32 { return r.Bottom - r.Top }

type msg struct {
	Hwnd     uintptr
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       point
	LPrivate uint32
}

type msllHookStruct struct {
	Pt          point
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type wndClassExW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type windowPlacement struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    point
	PtMaxPosition    point
	RcNormalPosition rect
}

type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

type notifyIconData struct {
	CbSize           uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UVersion         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

type keybdInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// keyboardInput mirrors INPUT with the keyboard member of the union selected.
type keyboardInput struct {
	Type uint32
	_    uint32
	Ki   keybdInput
	_    [8]byte // pad to sizeof(MOUSEINPUT), the largest union member
}

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		p, _ = syscall.UTF16PtrFromString("")
	}
	return p
}

func copyUTF16(dst []uint16, s string) {
	src, _ := syscall.UTF16FromString(s)
	if len(src) > len(dst) {
		src = src[:len(dst)]
		src[len(src)-1] = 0
	}
	copy(dst, src)
}

func moduleHandle() uintptr {
	h, _, _ := procGetModuleHandleW.Call(0)
	return h
}

// userUILanguage returns the LANGID of the user's Windows display language.
func userUILanguage() uint16 {
	r, _, _ := procGetUserDefaultUILanguage.Call()
	return uint16(r)
}

func keyDown(vk uintptr) bool {
	r, _, _ := procGetAsyncKeyState.Call(vk)
	return r&0x8000 != 0
}

func systemMetric(index uintptr) int32 {
	r, _, _ := procGetSystemMetrics.Call(index)
	return int32(r)
}

func windowFromPoint(pt point) uintptr {
	// POINT is passed by value, packed into a single register.
	r, _, _ := procWindowFromPoint.Call(uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32)
	return r
}

func rootWindow(hwnd uintptr) uintptr {
	r, _, _ := procGetAncestor.Call(hwnd, gaRoot)
	return r
}

func windowStyle(hwnd uintptr) uintptr {
	index := int32(gwlStyle)
	r, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(index))
	return r
}

func windowRect(hwnd uintptr) (rect, bool) {
	var r rect
	ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))
	return r, ok != 0
}

func isZoomed(hwnd uintptr) bool {
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}

func isHungAppWindow(hwnd uintptr) bool {
	r, _, _ := procIsHungAppWindow.Call(hwnd)
	return r != 0
}

func className(hwnd uintptr) string {
	var buf [256]uint16
	n, _, _ := procGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf[:n])
}

func windowProcessID(hwnd uintptr) (threadID, processID uint32) {
	t, _, _ := procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	return uint32(t), processID
}

// processName returns the executable name (e.g. "notepad.exe") of the process owning hwnd.
func processName(hwnd uintptr) string {
	_, pid := windowProcessID(hwnd)
	h, _, _ := procOpenProcess.Call(processQueryLimitedInformation, 0, uintptr(pid))
	if h == 0 {
		return ""
	}
	defer procCloseHandle.Call(h)
	var buf [1024]uint16
	n := uint32(len(buf))
	if ok, _, _ := procQueryFullProcessImageNameW.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n))); ok == 0 {
		return ""
	}
	return filepath.Base(syscall.UTF16ToString(buf[:n]))
}

func monitorRect(hwnd uintptr) rect {
	mon, _, _ := procMonitorFromWindow.Call(hwnd, monitorDefaultToNearest)
	mi := monitorInfo{}
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	return mi.RcMonitor
}

func normalRect(hwnd uintptr) rect {
	wp := windowPlacement{}
	wp.Length = uint32(unsafe.Sizeof(wp))
	procGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&wp)))
	return wp.RcNormalPosition
}

func foregroundWindow() uintptr {
	r, _, _ := procGetForegroundWindow.Call()
	return r
}

func messageBox(text string, flags uintptr) {
	procMessageBoxW.Call(0, uintptr(unsafe.Pointer(utf16Ptr(text))), uintptr(unsafe.Pointer(utf16Ptr(appName))), flags)
}

// sendMenuMaskKey taps an unassigned key so releasing a held Alt or Win key
// does not activate the target's menu bar or open the Start menu.
func sendMenuMaskKey() {
	in := [2]keyboardInput{
		{Type: inputKeyboard, Ki: keybdInput{WVk: vkMenuMask}},
		{Type: inputKeyboard, Ki: keybdInput{WVk: vkMenuMask, DwFlags: keyEventFKeyUp}},
	}
	procSendInput.Call(uintptr(len(in)), uintptr(unsafe.Pointer(&in[0])), unsafe.Sizeof(in[0]))
}

func enableDPIAwareness() {
	if procSetProcessDpiAwarenessContext.Find() == nil {
		if r, _, _ := procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2); r != 0 {
			return
		}
	}
	procSetProcessDPIAware.Call()
}
