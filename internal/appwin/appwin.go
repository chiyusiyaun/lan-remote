package appwin

import (
	_ "embed"
	"fmt"
	"os"
	"runtime"
)

//go:embed icon-client.ico
var IconClient []byte

//go:embed icon-server.ico
var IconServer []byte

// OpenWindow shows a native desktop window (WebView2 on Windows).
func OpenWindow(title, url string, w, h int) bool {
	return openWindow(title, url, w, h)
}

// RunWithTray opens a window and keeps a tray icon.
// icon may be nil (default). Use IconClient / IconServer.
func RunWithTray(title, url string, w, h int, startHidden bool, icon []byte) {
	runWithTray(title, url, w, h, startHidden, icon)
}

func Pause(msg string) { pause(msg) }

func OpenBrowser(url string) { openBrowser(url) }

func WaitSignal() { waitSignal() }

func Background(logPath string) { background(logPath) }

var _ = fmt.Println
var _ = os.Stdout
var _ = runtime.GOOS
