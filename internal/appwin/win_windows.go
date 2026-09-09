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
	shcore            = syscall.NewLazyDLL("shcore.dll")
	procShowWindow    = user32.NewProc("ShowWindow")
	procSetDpiAware   = shcore.NewProc("SetProcessDpiAwareness")
	procSetForeground = user32.NewProc("SetForegroundWindow")
	procIsIconic      = user32.NewProc("IsIconic")
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
	mu      sync.Mutex
	hwnd    uintptr
	url     string
	title   string
	w, h    int
	closed  bool
}

func (st *winState) show() {
	st.mu.Lock()
	h := st.hwnd
	st.mu.Unlock()
	if h == 0 {
		// window closed — reopen UI in browser (avoid second WebView on this thread)
		openBrowser(st.url)
		return
	}
	if v, _, _ := procIsIconic.Call(h); v != 0 {
		procShowWindow.Call(h, swRestore)
	}
	procShowWindow.Call(h, swShow)
	procSetForeground.Call(h)
}

func (st *winState) hide() {
	st.mu.Lock()
	h := st.hwnd
	st.mu.Unlock()
	if h != 0 {
		procShowWindow.Call(h, swHide)
	}
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

// runWithTray: Win32 tray on its own thread; WebView2 on main thread.
func runWithTray(title, url string, w, h int, startHidden bool, icon []byte) {
	st := &winState{url: url, title: title, w: w, h: h}

	// Tray first (separate message loop) — does not block WebView2.
	tray.Run(tray.Options{
		Tooltip: title,
		Icon:    icon,
		OnOpen:  st.show,
		OnHide:  st.hide,
		OnQuit:  func() { os.Exit(0) },
	})

	if startHidden {
		fmt.Println("Started hidden in tray. Use tray → 显示窗口.")
		waitSignal()
		return
	}

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
		openBrowser(url)
		waitSignal()
		return
	}
	defer wv.Destroy()

	if p := wv.Window(); p != nil {
		st.mu.Lock()
		st.hwnd = uintptr(p)
		st.mu.Unlock()
	}
	wv.Navigate(url)

	fmt.Println("Window + tray ready. Close window stays in tray.")
	wv.Run()

	// Window destroyed — keep control port in tray until 退出
	st.mu.Lock()
	st.hwnd = 0
	st.closed = true
	st.mu.Unlock()
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
