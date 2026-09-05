//go:build !windows

package appwin

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"

	"lan-remote/internal/tray"
)

func openWindow(title, url string, w, h int) bool { return false }

func runWithTray(title, url string, w, h int, startHidden bool) {
	// Linux: try tray if desktop session exists; always open browser for UI
	tray.Run(tray.Options{
		Tooltip: title,
		OnOpen:  func() { openBrowser(url) },
		OnQuit:  func() { os.Exit(0) },
	})
	openBrowser(url)
	fmt.Println("Running with tray (if available). Ctrl+C to exit.")
	waitSignal()
}

func openBrowser(url string) {
	if runtime.GOOS == "darwin" {
		_ = exec.Command("open", url).Start()
		return
	}
	_ = exec.Command("xdg-open", url).Start()
}

func pause(msg string) {
	fmt.Println(msg)
	fmt.Println("Press Enter to exit...")
	var b [1]byte
	_, _ = os.Stdin.Read(b[:])
}

func waitSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}

func background(logPath string) {
	if logPath == "" {
		logPath = "/tmp/lan-remote.log"
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		os.Stdout = f
		os.Stderr = f
	}
}
