//go:build windows

package appwin

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	webview2 "github.com/jchv/go-webview2"
)

func openWindow(title, url string, w, h int) bool {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("webview panic:", r)
		}
	}()
	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  title,
			Width:  uint(w),
			Height: uint(h),
			Center: true,
		},
	})
	if wv == nil {
		return false
	}
	defer wv.Destroy()
	wv.Navigate(url)
	wv.Run()
	return true
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func waitSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}
