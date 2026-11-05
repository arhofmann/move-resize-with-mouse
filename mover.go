package main

import (
	"runtime"
	"sync"
)

// windowOp is one change to apply to a foreign window.
type windowOp struct {
	hwnd     uintptr
	restore  bool // un-maximize first
	activate bool // bring to the foreground
	setRect  bool
	resize   bool // apply rect's size too, not just its position
	rect     rect
	// Edges that must stay where rect puts them when Windows adjusts the size.
	keepRight, keepBottom bool
}

func (op windowOp) geometryOnly() bool { return !op.restore && !op.activate }

// mover applies the window operations of one drag on its own thread.
// SetWindowPos on another process's window blocks until that window has
// handled it; doing that here keeps the mouse hook responsive and lets pending
// moves collapse into one when the target repaints slower than the mouse
// moves. Every drag gets a fresh mover, so a window that hangs only stalls its
// own drag, not later drags of other windows.
type mover struct {
	mu    sync.Mutex
	queue []windowOp
	wake  chan struct{}
	done  chan struct{} // closed once everything submitted has been applied
}

// movers holds the latest mover per window (hook thread only), so a new drag
// of a window waits until the previous drag's operations have been applied.
var movers = map[uintptr]*mover{}

// newMover starts the mover for a drag of hwnd.
func newMover(hwnd uintptr) *mover {
	for h, m := range movers {
		select {
		case <-m.done:
			delete(movers, h)
		default:
		}
	}
	m := &mover{wake: make(chan struct{}, 1), done: make(chan struct{})}
	go m.run(movers[hwnd])
	movers[hwnd] = m
	return m
}

func (m *mover) submit(op windowOp) {
	m.mu.Lock()
	n := len(m.queue)
	if n > 0 && op.geometryOnly() {
		last := m.queue[n-1]
		if last.geometryOnly() && last.hwnd == op.hwnd && last.resize == op.resize {
			m.queue[n-1] = op
			n = -1
		}
	}
	if n >= 0 {
		m.queue = append(m.queue, op)
	}
	m.mu.Unlock()

	select {
	case m.wake <- struct{}{}:
	default:
	}
}

// finish lets the mover exit once it has applied everything submitted so far.
// Nothing may be submitted afterwards.
func (m *mover) finish() { close(m.wake) }

func (m *mover) run(prev *mover) {
	runtime.LockOSThread() // AttachThreadInput is per OS thread; the thread ends with the mover
	defer close(m.done)
	if prev != nil {
		<-prev.done
	}
	for range m.wake {
		m.mu.Lock()
		ops := m.queue
		m.queue = nil
		m.mu.Unlock()
		for _, op := range ops {
			op.apply()
		}
	}
}

func (op windowOp) apply() {
	if op.restore {
		procShowWindow.Call(op.hwnd, swRestore)
	}
	if op.activate {
		activateWindow(op.hwnd)
	}
	if op.setRect {
		flags := uintptr(swpNoZOrder | swpNoActivate | swpNoOwnerZOrder)
		if !op.resize {
			flags |= swpNoSize
		}
		r := op.rect
		procSetWindowPos.Call(op.hwnd, 0,
			uintptr(r.Left), uintptr(r.Top), uintptr(r.width()), uintptr(r.height()), flags)
		if op.keepRight || op.keepBottom {
			// Windows enforces the window's own minimum and maximum size but
			// keeps the requested position, which would shove the edge that is
			// meant to stay put. Move the window back so that edge stays.
			if got, ok := windowRect(op.hwnd); ok && (got.width() != r.width() || got.height() != r.height()) {
				x, y := got.Left, got.Top
				if op.keepRight {
					x = r.Right - got.width()
				}
				if op.keepBottom {
					y = r.Bottom - got.height()
				}
				procSetWindowPos.Call(op.hwnd, 0, uintptr(x), uintptr(y), 0, 0, flags|swpNoSize)
			}
		}
	}
}

// activateWindow brings hwnd to the foreground, working around the foreground
// lock that normally stops background processes from doing so.
func activateWindow(hwnd uintptr) {
	if foregroundWindow() == hwnd {
		return
	}
	procSetForegroundWindow.Call(hwnd)
	fg := foregroundWindow()
	if fg == hwnd {
		return
	}
	fgThread, _ := windowProcessID(fg)
	self, _, _ := procGetCurrentThreadId.Call()
	if fgThread == 0 || uintptr(fgThread) == self {
		procBringWindowToTop.Call(hwnd)
		return
	}
	procAttachThreadInput.Call(self, uintptr(fgThread), 1)
	procSetForegroundWindow.Call(hwnd)
	procBringWindowToTop.Call(hwnd)
	procAttachThreadInput.Call(self, uintptr(fgThread), 0)
}
