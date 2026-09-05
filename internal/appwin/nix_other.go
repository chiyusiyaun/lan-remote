//go:build !windows

package appwin

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

func openWindow(title, url string, w, h int) bool { return false }

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
