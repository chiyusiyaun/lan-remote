//go:build windows

package appwin

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	webview2 "github.com/jchv/go-webview2"

	"lan-remote/internal/tray"
)

var (
	shcore             = syscall.NewLazyDLL("shcore.dll")
	procShowWindow     = user32.NewProc("ShowWindow")
	procSetDpiAware    = shcore.NewProc("SetProcessDpiAwareness")
)

const (
	swHide    = 0
	swShow    = 5
	swRestore = 9
)

func init() {
	// PROCESS_PER_MONITOR_DPI_AWARE = 2
	_, _, _ = procSetDpiAware.Call(2)
}

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
	fmt.Println("Native app window opened.")
	wv.Run()
	return true
}

func runWithTray(title, url string, w, h int, startHidden bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("tray window panic:", r)
		}
	}()

	holder := &struct{ hwnd uintptr }{}

	show := func() {
		if holder.hwnd != 0 {
			procShowWindow.Call(holder.hwnd, swShow)
			procShowWindow.Call(holder.hwnd, swRestore)
		}
	}
	hide := func() {
		if holder.hwnd != 0 {
			procShowWindow.Call(holder.hwnd, swHide)
		}
	}
	quit := func() { os.Exit(0) }

	tray.Run(tray.Options{
		Tooltip: title,
		OnOpen:  show,
		OnHide:  hide,
		OnQuit:  quit,
	})

	wv := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: !startHidden,
		WindowOptions: webview2.WindowOptions{
			Title:  title,
			Width:  uint(w),
			Height: uint(h),
			Center: true,
		},
	})
	if wv == nil {
		openBrowser(url)
		waitSignal()
		return
	}
	defer wv.Destroy()
	if p := wv.Window(); p != nil {
		holder.hwnd = uintptr(p)
	}
	wv.Navigate(url)
	if startHidden {
		hide()
	} else {
		show()
	}
	fmt.Println("Window + tray ready. Close window stays in tray; quit from tray menu.")
	wv.Run()
	waitSignal()
}

func openBrowser(url string) {
	_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}

func waitSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}

func background(logPath string) {
	if logPath == "" {
		logPath = os.Getenv("TEMP") + `\lan-remote.log`
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err == nil {
		os.Stdout = f
		os.Stderr = f
	}
	k32 := syscall.NewLazyDLL("kernel32.dll")
	proc := k32.NewProc("FreeConsole")
	_, _, _ = proc.Call()
}
