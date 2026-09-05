//go:build windows

package appwin

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	webview2 "github.com/jchv/go-webview2"

	"lan-remote/internal/tray"
)

var (
	shcore          = syscall.NewLazyDLL("shcore.dll")
	procShowWindow  = user32.NewProc("ShowWindow")
	procSetDpiAware = shcore.NewProc("SetProcessDpiAwareness")
	procSetForeground = user32.NewProc("SetForegroundWindow")
	procIsIconic    = user32.NewProc("IsIconic")
)

const (
	swHide    = 0
	swShow    = 5
	swRestore = 9
)

func init() {
	_, _, _ = procSetDpiAware.Call(2)
}

type winState struct {
	mu   sync.Mutex
	wv   webview2.WebView
	hwnd uintptr
	url  string
	title string
	w, h int
	opening bool
}

func (st *winState) showExisting() bool {
	st.mu.Lock()
	h := st.hwnd
	st.mu.Unlock()
	if h == 0 {
		return false
	}
	// restore if minimized, then show + foreground
	if v, _, _ := procIsIconic.Call(h); v != 0 {
		procShowWindow.Call(h, swRestore)
	}
	procShowWindow.Call(h, swShow)
	procShowWindow.Call(h, swRestore)
	procSetForeground.Call(h)
	return true
}

func (st *winState) hide() {
	st.mu.Lock()
	h := st.hwnd
	st.mu.Unlock()
	if h != 0 {
		procShowWindow.Call(h, swHide)
	}
}

func (st *winState) openNew() {
	st.mu.Lock()
	if st.opening || st.wv != nil {
		st.mu.Unlock()
		st.showExisting()
		return
	}
	st.opening = true
	st.mu.Unlock()

	go func() {
		defer func() {
			st.mu.Lock()
			st.opening = false
			st.mu.Unlock()
			if r := recover(); r != nil {
				fmt.Println("window panic:", r)
			}
		}()

		wv := webview2.NewWithOptions(webview2.WebViewOptions{
			AutoFocus: true,
			WindowOptions: webview2.WindowOptions{
				Title:  st.title,
				Width:  uint(st.w),
				Height: uint(st.h),
				Center: true,
			},
		})
		if wv == nil {
			openBrowser(st.url)
			return
		}

		var hwnd uintptr
		if p := wv.Window(); p != nil {
			hwnd = uintptr(p)
		}
		st.mu.Lock()
		st.wv = wv
		st.hwnd = hwnd
		st.mu.Unlock()

		wv.Navigate(st.url)
		procShowWindow.Call(hwnd, swShow)
		procSetForeground.Call(hwnd)

		wv.Run() // blocks until window destroyed

		st.mu.Lock()
		st.wv = nil
		st.hwnd = 0
		st.mu.Unlock()
		// window already closed; swallow destroy errors
		func() {
			defer func() { _ = recover() }()
			wv.Destroy()
		}()
	}()
}

func (st *winState) showOrOpen() {
	if st.showExisting() {
		return
	}
	st.openNew()
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
	st := &winState{url: url, title: title, w: w, h: h}

	tray.Run(tray.Options{
		Tooltip: title,
		OnOpen:  st.showOrOpen,
		OnHide:  st.hide,
		OnQuit:  func() { os.Exit(0) },
	})

	if startHidden {
		// stay in tray; user opens from menu
		fmt.Println("Started hidden in tray. Use tray icon → 显示窗口.")
	} else {
		st.openNew()
	}

	// keep process alive until quit
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
