# EasyMove+Resize for Windows

Move and resize any window by holding **Alt** and dragging anywhere inside it, with
no need to aim for the title bar or a border. A Windows take on
[Easy Move+Resize](https://github.com/dmarcotte/easy-move-resize) for macOS.

| Action | Effect |
| --- | --- |
| **Alt** + left-drag | Move the window |
| **Alt** + right-drag | Resize the window |

The modifier key (or combination) and the resize button can be changed from the tray icon.

## Install

Download `EasyMoveResize.exe` from the [latest release](../../releases/latest) and run it.
There is no installer and nothing else to install: the app is a single ~3 MB executable
that lives in the notification area of the taskbar. To have it start automatically,
right-click the tray icon and choose *Start with Windows*.

Requires 64-bit Windows 10 or 11.

> [!NOTE]
> The executable is not code-signed, so Windows SmartScreen may warn about it the first
> time. Click *More info* → *Run anyway*, or [build it yourself](#building).

## How it works

- **Resizing** splits the window into a 3×3 grid. Grab near a corner to drag that corner,
  near the middle of an edge to drag just that edge. Grabbing the very center resizes from
  the bottom-right corner.
- **Maximized windows** are restored to their normal size as soon as you start dragging
  them. A plain Alt+click leaves them maximized.
- Pressing **another mouse button** while dragging cancels the drag.
- **Fullscreen apps** (games, video players), the desktop, the taskbar and the Start menu
  are left alone, as are windows that are not responding.
- While the modifier is held, clicks that start a drag are not passed on to the window
  underneath, so Alt+click won't also click a button or select text.

## Tray menu

Left-click the tray icon (a white four-way arrow on blue) to turn the app on or off; it
turns gray while off. Right-click it for settings:

| Menu item | Meaning |
| --- | --- |
| *Enabled* | On/off |
| *Modifier* | Modifier key(s): any combination of Ctrl, Alt, Shift and Win |
| *Bring window to front* | Bring the dragged window to the front |
| *Resize with middle button* | Resize with the middle button instead of the right one |
| *Excluded programs* | Programs to ignore, e.g. ones that use Alt+click themselves. After a drag, the program you just dragged is offered here (*Exclude …*) |
| *Start with Windows* | Start automatically when you sign in |
| *Exit* | Quit the app |

The menu and messages are in German if your Windows display language is German, and in
English otherwise.

Some programs use Alt+click for their own features (multiple cursors in VS Code, camera
controls in Blender, …). Exclude them, or switch to a different modifier.

## Configuration

Settings are stored in `%APPDATA%\EasyMoveResize\config.json` and written whenever you
change something in the tray menu. To edit the file by hand, quit the app first:

```json
{
  "enabled": true,
  "modifiers": ["alt"],
  "bringToFront": true,
  "middleClickResize": false,
  "excludedApps": ["Code.exe", "blender.exe"]
}
```

- `modifiers`: any of `"ctrl"`, `"alt"`, `"shift"`, `"win"`. All listed keys must be held.
- `excludedApps`: executable file names, matched case-insensitively.

## Limitations

- Windows only lets a program move windows of programs running as administrator if it is
  itself running as administrator. Start EasyMove+Resize elevated if you need that.
- Releasing Alt or Win normally opens the window menu or the Start menu. To prevent that,
  the app sends an unassigned key (VK 0xE8) when a drag starts, the same trick AutoHotkey uses.

## Building

Requires [Go](https://go.dev/dl/) 1.22 or later. There are no dependencies beyond the
standard library: Win32 is called directly through `syscall`.

```powershell
.\build.ps1
```

or

```powershell
go build -trimpath -ldflags "-H windowsgui -s -w" -o EasyMoveResize.exe .
```

`-H windowsgui` keeps a console window from opening. The code uses no cgo, so it can also be
cross-compiled from Linux or macOS with `GOOS=windows GOARCH=amd64`.

To publish a release, push a tag such as `v1.0.0`. GitHub Actions then builds the executable
and attaches it, with its SHA-256 checksum, to a new GitHub release.

## Project layout

| File | Contents |
| --- | --- |
| `hook.go` | Global low-level mouse hook (`WH_MOUSE_LL`) on its own thread; starting, tracking and ending drags |
| `mover.go` | One thread per drag that calls `SetWindowPos`, coalescing mouse moves |
| `tray.go` | Hidden window, tray icon and menu |
| `config.go` | Settings (JSON) and autostart (registry `HKCU\…\Run`) |
| `strings.go` | User-facing text in English and German |
| `icon.go` | Draws the tray icon at runtime |
| `win32.go` | Minimal Win32 bindings via `syscall` |

## Credits

Inspired by [Easy Move+Resize](https://github.com/dmarcotte/easy-move-resize) for macOS by
Daniel Marcotte. This is an independent reimplementation for Windows and is not affiliated
with that project.

## License

[MIT](LICENSE)
