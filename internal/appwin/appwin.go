package appwin

import (
	"fmt"
	"os"
	"runtime"
)

// OpenWindow shows a native desktop window (WebView2 on Windows).
// Returns false if unavailable.
func OpenWindow(title, url string, w, h int) bool {
	return openWindow(title, url, w, h)
}

// Pause shows an error dialog / waits for Enter.
func Pause(msg string) {
	pause(msg)
}

func OpenBrowser(url string) {
	openBrowser(url)
}

func WaitSignal() {
	waitSignal()
}

// silence
var _ = fmt.Println
var _ = os.Stdout
var _ = runtime.GOOS
