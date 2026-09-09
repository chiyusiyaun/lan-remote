//go:build !windows

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

func Available() bool {
	if runtime.GOOS == "darwin" {
		return true
	}
	return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
}

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
}

func onExit() {}

func Quit() {
	if !Available() {
		return
	}
	systray.Quit()
}
