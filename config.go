package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"
)

const (
	modAlt uint32 = 1 << iota
	modCtrl
	modShift
	modWin
)

// modifiers lists the supported modifier keys in display order.
var modifiers = []struct {
	bit   uint32
	key   string // name in config.json
	label string // name in the tray menu
}{
	{modCtrl, "ctrl", ui.ctrl},
	{modAlt, "alt", "Alt"},
	{modShift, "shift", ui.shift},
	{modWin, "win", "Win"},
}

type config struct {
	Enabled           bool     `json:"enabled"`
	Modifiers         []string `json:"modifiers"`
	BringToFront      bool     `json:"bringToFront"`
	MiddleClickResize bool     `json:"middleClickResize"`
	ExcludedApps      []string `json:"excludedApps"`

	mods uint32 // Modifiers as a bit mask
}

var cfg *config // UI thread only

// hookConfig is the copy of cfg the mouse hook thread works with; the UI
// thread replaces it whenever a setting changes.
var hookConfig atomic.Pointer[config]

func (c *config) publish() {
	cp := *c
	cp.ExcludedApps = slices.Clone(c.ExcludedApps)
	hookConfig.Store(&cp)
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Dir(os.Args[0])
	}
	return filepath.Join(dir, "EasyMoveResize", "config.json")
}

// loadConfig reads the config file, falling back to defaults, and hands the
// result to the mouse hook. firstRun reports whether no config file existed yet.
func loadConfig() (c *config, firstRun bool) {
	c = &config{
		Enabled:      true,
		Modifiers:    []string{"alt"},
		BringToFront: true,
		ExcludedApps: []string{},
	}
	data, err := os.ReadFile(configPath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
		firstRun = true
	case err == nil:
		_ = json.Unmarshal(data, c)
	}
	for _, name := range c.Modifiers {
		for _, m := range modifiers {
			if strings.EqualFold(name, m.key) {
				c.mods |= m.bit
			}
		}
	}
	if c.mods == 0 {
		c.mods = modAlt
	}
	if c.ExcludedApps == nil {
		c.ExcludedApps = []string{}
	}
	if firstRun {
		c.save()
	}
	c.publish()
	return c, firstRun
}

// save hands the settings to the mouse hook and writes them to disk.
func (c *config) save() {
	c.publish()
	c.Modifiers = c.Modifiers[:0]
	for _, m := range modifiers {
		if c.mods&m.bit != 0 {
			c.Modifiers = append(c.Modifiers, m.key)
		}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

// modifierLabel renders the configured modifiers, e.g. "Ctrl+Alt".
func (c *config) modifierLabel() string {
	var parts []string
	for _, m := range modifiers {
		if c.mods&m.bit != 0 {
			parts = append(parts, m.label)
		}
	}
	return strings.Join(parts, "+")
}

func (c *config) isExcluded(app string) bool {
	for _, e := range c.ExcludedApps {
		if strings.EqualFold(e, app) {
			return true
		}
	}
	return false
}

func (c *config) setExcluded(app string, excluded bool) {
	kept := c.ExcludedApps[:0]
	for _, e := range c.ExcludedApps {
		if !strings.EqualFold(e, app) {
			kept = append(kept, e)
		}
	}
	c.ExcludedApps = kept
	if excluded {
		c.ExcludedApps = append(c.ExcludedApps, app)
	}
	c.save()
}

const (
	runKey   = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue = "EasyMoveResize"
)

func autostartEnabled() bool {
	r, _, _ := procRegGetValueW.Call(hkeyCurrentUser,
		uintptr(unsafe.Pointer(utf16Ptr(runKey))), uintptr(unsafe.Pointer(utf16Ptr(runValue))),
		rrfRtRegSz, 0, 0, 0)
	return r == 0
}

func setAutostart(on bool) {
	if !on {
		procRegDeleteKeyValueW.Call(hkeyCurrentUser,
			uintptr(unsafe.Pointer(utf16Ptr(runKey))), uintptr(unsafe.Pointer(utf16Ptr(runValue))))
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	data, _ := syscall.UTF16FromString(`"` + exe + `"`)
	procRegSetKeyValueW.Call(hkeyCurrentUser,
		uintptr(unsafe.Pointer(utf16Ptr(runKey))), uintptr(unsafe.Pointer(utf16Ptr(runValue))),
		regSz, uintptr(unsafe.Pointer(&data[0])), uintptr(len(data)*2))
}
