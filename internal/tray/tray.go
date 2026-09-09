package tray

import (
	"log"
	"os"
	"runtime"
	"sync"

	"github.com/energye/systray"
)

type Options struct {
	Tooltip string
	Icon    []byte
	OnOpen  func()
	OnHide  func()
	OnQuit  func()
}

var (
	mu       sync.Mutex
	started  bool
	optsKeep Options
)

// Available reports whether a desktop tray can run on this session.
func Available() bool {
	if runtime.GOOS == "windows" {
		return true
	}
	// Linux/macOS: need a graphical session
	if os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return true
	}
	if runtime.GOOS == "darwin" {
		return true
	}
	return false
}

// Run starts the tray icon (non-blocking). No-op if no desktop session.
func Run(opts Options) {
	if !Available() {
		log.Println("tray: skipped (no desktop session)")
		return
	}
	mu.Lock()
	if started {
		mu.Unlock()
		return
	}
	started = true
	optsKeep = opts
	mu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Println("tray:", r)
			}
		}()
		systray.Run(onReady, onExit)
	}()
}

func onReady() {
	title := optsKeep.Tooltip
	if title == "" {
		title = "LAN Remote"
	}
	systray.SetTitle("LR")
	systray.SetTooltip(title)
	if len(optsKeep.Icon) > 0 {
		systray.SetIcon(optsKeep.Icon)
	} else {
		systray.SetIcon(iconBytes)
	}

	mOpen := systray.AddMenuItem("显示窗口", "Show window")
	mHide := systray.AddMenuItem("隐藏到托盘", "Hide to tray")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "Quit")

	mOpen.Click(func() {
		if optsKeep.OnOpen != nil {
			optsKeep.OnOpen()
		}
	})
	mHide.Click(func() {
		if optsKeep.OnHide != nil {
			optsKeep.OnHide()
		}
	})
	mQuit.Click(func() {
		if optsKeep.OnQuit != nil {
			optsKeep.OnQuit()
		}
		systray.Quit()
	})
	systray.SetOnDClick(func(menu systray.IMenu) {
		if optsKeep.OnOpen != nil {
			optsKeep.OnOpen()
		}
	})
}

func onExit() {}

func Quit() {
	if !Available() {
		return
	}
	systray.Quit()
}
