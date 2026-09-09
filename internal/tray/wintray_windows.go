//go:build windows

package tray

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClass      = user32.NewProc("RegisterClassW")
	procCreateWindowEx     = user32.NewProc("CreateWindowExW")
	procDefWindowProc      = user32.NewProc("DefWindowProcW")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procGetMessage         = user32.NewProc("GetMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procDispatchMessage    = user32.NewProc("DispatchMessageW")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procCreatePopupMenu    = user32.NewProc("CreatePopupMenu")
	procAppendMenu         = user32.NewProc("AppendMenuW")
	procTrackPopupMenu     = user32.NewProc("TrackPopupMenu")
	procGetCursorPos       = user32.NewProc("GetCursorPos")
	procSetForeground      = user32.NewProc("SetForegroundWindow")
	procDestroyMenu        = user32.NewProc("DestroyMenu")
	procGetModuleHandle    = kernel32.NewProc("GetModuleHandleW")
	procShellNotifyIcon    = shell32.NewProc("Shell_NotifyIconW")
	procLoadIcon           = user32.NewProc("LoadIconW")
	procLoadCursor         = user32.NewProc("LoadCursorW")
)

const (
	wmUser        = 0x0400
	wmTray        = wmUser + 1
	wmDestroy     = 0x0002
	wmCommand     = 0x0111
	wmLButtonUp   = 0x0202
	wmRButtonUp   = 0x0205
	wmLButtonDbl  = 0x0203
	nimAdd        = 0x00000000
	nimDelete     = 0x00000002
	nimModify     = 0x00000001
	nifMessage    = 0x00000001
	nifIcon       = 0x00000002
	nifTip        = 0x00000004
	mfString      = 0x00000000
	mfSeparator   = 0x00000800
	idcArrow      = 32512
	idiApplication = 32512
	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020
	tpmLeftAlign   = 0x0000
	hwndMessage    = ^uintptr(0) - 2 // HWND_MESSAGE
	csHRedraw      = 0x0002
	csVRedraw      = 0x0001
	wsOverlapped   = 0
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type point struct{ X, Y int32 }

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type notifyIconData struct {
	Size            uint32
	Wnd             uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	TimeoutOrVersion uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GUID            [16]byte
}

type Options struct {
	Tooltip string
	Icon    []byte
	OnOpen  func()
	OnHide  func()
	OnQuit  func()
}

var (
	hwnd      uintptr
	optsKeep  Options
	running   bool
)

// Available — always true on Windows.
func Available() bool { return true }

// Run starts a Win32 tray icon on its own message loop (safe with WebView2).
func Run(opts Options) {
	if running {
		return
	}
	running = true
	optsKeep = opts
	go trayLoop()
}

func trayLoop() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("tray:", r)
		}
	}()

	hInst, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("LANRemoteTrayWnd")
	wndProc := syscall.NewCallback(trayWndProc)

	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:   wndProc,
		Instance:  hInst,
		ClassName: className,
	}
	procRegisterClass.Call(uintptr(unsafe.Pointer(&wc)))

	hwnd, _, _ = procCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0, 0, 0, 0, 0, 0,
		hwndMessage,
		0, hInst, 0,
	)
	if hwnd == 0 {
		fmt.Println("tray: CreateWindowEx failed")
		return
	}

	addTrayIcon()
	defer delTrayIcon()

	var m msg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func trayWndProc(h, msgU, wParam, lParam uintptr) uintptr {
	switch msgU {
	case wmTray:
		switch lParam {
		case wmLButtonUp, wmLButtonDbl:
			if optsKeep.OnOpen != nil {
				optsKeep.OnOpen()
			}
		case wmRButtonUp:
			showMenu()
		}
		return 0
	case wmCommand:
		id := wParam & 0xFFFF
		switch id {
		case 1:
			if optsKeep.OnOpen != nil {
				optsKeep.OnOpen()
			}
		case 2:
			if optsKeep.OnHide != nil {
				optsKeep.OnHide()
			}
		case 3:
			if optsKeep.OnQuit != nil {
				optsKeep.OnQuit()
			}
			procPostQuitMessage.Call(0)
		}
		return 0
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(h, msgU, wParam, lParam)
	return ret
}

func addTrayIcon() {
	var nid notifyIconData
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Wnd = hwnd
	nid.ID = 1
	nid.Flags = nifMessage | nifIcon | nifTip
	nid.CallbackMessage = wmTray
	// default app icon
	nid.Icon, _, _ = procLoadIcon.Call(0, idiApplication)
	tip := optsKeep.Tooltip
	if tip == "" {
		tip = "LAN Remote"
	}
	copyTip := syscall.StringToUTF16(tip)
	for i := 0; i < len(nid.Tip)-1 && i < len(copyTip)-1; i++ {
		nid.Tip[i] = copyTip[i]
	}
	procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

func delTrayIcon() {
	if hwnd == 0 {
		return
	}
	var nid notifyIconData
	nid.Size = uint32(unsafe.Sizeof(nid))
	nid.Wnd = hwnd
	nid.ID = 1
	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func showMenu() {
	menu, _, _ := procCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer procDestroyMenu.Call(menu)

	open, _ := syscall.UTF16PtrFromString("显示窗口")
	hide, _ := syscall.UTF16PtrFromString("隐藏到托盘")
	quit, _ := syscall.UTF16PtrFromString("退出")
	procAppendMenu.Call(menu, mfString, 1, uintptr(unsafe.Pointer(open)))
	procAppendMenu.Call(menu, mfString, 2, uintptr(unsafe.Pointer(hide)))
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
	procAppendMenu.Call(menu, mfString, 3, uintptr(unsafe.Pointer(quit)))

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForeground.Call(hwnd)
	procTrackPopupMenu.Call(menu, tpmRightButton|tpmBottomAlign|tpmLeftAlign,
		uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
}

func Quit() {
	if hwnd != 0 {
		procDestroyWindow.Call(hwnd)
	}
}
