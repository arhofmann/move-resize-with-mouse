package main

// User-facing text: English by default, German if the Windows display language
// is German.

type uiText struct {
	// Modifier key names; Alt and Win are the same in both languages.
	ctrl, shift string

	alreadyRunning string
	startFailed    string // followed by the error
	firstRunTitle  string
	firstRunText   string // %[1]s: modifier keys

	tipDisabled     string
	tipEnabled      string // %[1]s: modifier keys, %[2]s: tipRightButton or tipMiddleButton
	tipRightButton  string
	tipMiddleButton string

	menuEnabled      string
	menuModifier     string // followed by the modifier keys
	menuBringToFront string
	menuMiddleResize string
	menuExcluded     string
	menuExcludedNone string
	menuExclude      string // %s: program name
	menuAutostart    string
	menuQuit         string
}

var english = uiText{
	ctrl:  "Ctrl",
	shift: "Shift",

	alreadyRunning: appName + " is already running (see the notification area of the taskbar).",
	startFailed:    appName + " could not start:\n",
	firstRunTitle:  appName + " is running",
	firstRunText: "%[1]s + left-drag: move a window\n" +
		"%[1]s + right-drag: resize a window\n" +
		"Right-click this icon for settings.",

	tipDisabled:     appName + " (disabled)",
	tipEnabled:      appName + "\n%[1]s + left-drag: move\n%[1]s + %[2]s: resize",
	tipRightButton:  "right-drag",
	tipMiddleButton: "middle-drag",

	menuEnabled:      "Enabled",
	menuModifier:     "Modifier: ",
	menuBringToFront: "Bring window to front",
	menuMiddleResize: "Resize with middle button",
	menuExcluded:     "Excluded programs",
	menuExcludedNone: "(none)",
	menuExclude:      "Exclude %s",
	menuAutostart:    "Start with Windows",
	menuQuit:         "Exit",
}

var german = uiText{
	ctrl:  "Strg",
	shift: "Umschalt",

	alreadyRunning: appName + " läuft bereits (siehe Infobereich der Taskleiste).",
	startFailed:    appName + " konnte nicht starten:\n",
	firstRunTitle:  appName + " läuft",
	firstRunText: "%[1]s + linke Maustaste: Fenster verschieben\n" +
		"%[1]s + rechte Maustaste: Größe ändern\n" +
		"Rechtsklick auf dieses Symbol für Einstellungen.",

	tipDisabled:     appName + " (deaktiviert)",
	tipEnabled:      appName + "\n%[1]s + Linksklick: Verschieben\n%[1]s + %[2]s: Größe ändern",
	tipRightButton:  "Rechtsklick",
	tipMiddleButton: "Mittelklick",

	menuEnabled:      "Aktiviert",
	menuModifier:     "Taste: ",
	menuBringToFront: "Fenster in den Vordergrund holen",
	menuMiddleResize: "Größe mit mittlerer Maustaste ändern",
	menuExcluded:     "Ausgeschlossene Programme",
	menuExcludedNone: "(keine)",
	menuExclude:      "%s ausschließen",
	menuAutostart:    "Mit Windows starten",
	menuQuit:         "Beenden",
}

// ui is the text for the user's Windows display language. It is set during
// package initialization, before modifiers picks up its key names.
var ui = uiTextForLanguage(userUILanguage())

func uiTextForLanguage(langID uint16) *uiText {
	const langGerman = 0x07
	if langID&0x3FF == langGerman { // PRIMARYLANGID
		return &german
	}
	return &english
}
