package appwin

import (
	"fmt"
	"os"
	"runtime"
)

// OpenWindow shows a native desktop window (WebView2 on Windows).
func OpenWindow(title, url string, w, h int) bool {
	return openWindow(title, url, w, h)
}

// RunWithTray opens a window and keeps a tray icon (Windows; best-effort on Linux).
// onHide/show control the main window. onQuit exits the process.
func RunWithTray(title, url string, w, h int, startHidden bool) {
	runWithTray(title, url, w, h, startHidden)
}

func Pause(msg string) { pause(msg) }

func OpenBrowser(url string) { openBrowser(url) }

func WaitSignal() { waitSignal() }

// Background prepares a headless/daemon-friendly run (log file, no console).
func Background(logPath string) {
	background(logPath)
}

var _ = fmt.Println
var _ = os.Stdout
var _ = runtime.GOOS
