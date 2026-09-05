//go:build windows

package inject

import (
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procSendInput        = user32.NewProc("SendInput")
	procSetCursorPos     = user32.NewProc("SetCursorPos")
	procVkKeyScanW       = user32.NewProc("VkKeyScanW")
	procMapVirtualKeyW   = user32.NewProc("MapVirtualKeyW")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const (
	inputMouse    = 0
	inputKeyboard = 1

	mouseeventfLeftDown   = 0x0002
	mouseeventfLeftUp     = 0x0004
	mouseeventfRightDown  = 0x0008
	mouseeventfRightUp    = 0x0010
	mouseeventfMiddleDown = 0x0020
	mouseeventfMiddleUp   = 0x0040
	mouseeventfWheel      = 0x0800
	mouseeventfHWheel     = 0x1000

	keyeventfKeyup   = 0x0002
	keyeventfUnicode = 0x0004

	wheelDelta = 120
	smCXScreen = 0
	smCYScreen = 1
)

// Windows INPUT is 40 bytes on amd64.
// MOUSEINPUT payload is 32 bytes; KEYBDINPUT is 24 + 8 pad.
type mouseInput struct {
	Dx, Dy    int32
	MouseData uint32
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type keybdInput struct {
	Vk, Scan  uint16
	Flags     uint32
	Time      uint32
	ExtraInfo uintptr
}

type winInputMouse struct {
	Type uint32
	_    uint32
	Mi   mouseInput
}

type winInputKey struct {
	Type uint32
	_    uint32
	Ki   keybdInput
	_    uint64 // pad union to 32 bytes so sizeof == 40
}

type windowsController struct{}

func newController() Controller { return &windowsController{} }

func (c *windowsController) MouseMove(x, y int) {
	procSetCursorPos.Call(uintptr(x), uintptr(y))
}

func sendMouse(flags, data uint32) {
	in := winInputMouse{Type: inputMouse}
	in.Mi.Flags = flags
	in.Mi.MouseData = data
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func (c *windowsController) MouseButton(btn string, down bool) {
	var flags uint32
	switch btn {
	case "left":
		if down {
			flags = mouseeventfLeftDown
		} else {
			flags = mouseeventfLeftUp
		}
	case "right":
		if down {
			flags = mouseeventfRightDown
		} else {
			flags = mouseeventfRightUp
		}
	case "middle":
		if down {
			flags = mouseeventfMiddleDown
		} else {
			flags = mouseeventfMiddleUp
		}
	default:
		return
	}
	sendMouse(flags, 0)
}

func (c *windowsController) MouseScroll(dx, dy int) {
	if dy != 0 {
		sendMouse(mouseeventfWheel, uint32(uint32(int32(dy*wheelDelta))))
	}
	if dx != 0 {
		sendMouse(mouseeventfHWheel, uint32(uint32(int32(dx*wheelDelta))))
	}
}

var vkMap = map[string]uint16{
	"Backspace": 0x08, "Tab": 0x09, "Enter": 0x0D, "Shift": 0x10, "Control": 0x11,
	"Alt": 0x12, "Pause": 0x13, "CapsLock": 0x14, "Escape": 0x1B, "Space": 0x20,
	"PageUp": 0x21, "PageDown": 0x22, "End": 0x23, "Home": 0x24, "Left": 0x25,
	"ArrowLeft": 0x25, "Up": 0x26, "ArrowUp": 0x26, "Right": 0x27, "ArrowRight": 0x27,
	"Down": 0x28, "ArrowDown": 0x28, "Insert": 0x2D, "Delete": 0x2E,
	"Meta": 0x5B, "ContextMenu": 0x5D, "NumLock": 0x90, "ScrollLock": 0x91,
	"F1": 0x70, "F2": 0x71, "F3": 0x72, "F4": 0x73, "F5": 0x74, "F6": 0x75,
	"F7": 0x76, "F8": 0x77, "F9": 0x78, "F10": 0x79, "F11": 0x7A, "F12": 0x7B,
}

func (c *windowsController) Key(key string, down bool) {
	vk, ok := vkMap[key]
	if !ok {
		r := []rune(key)
		if len(r) != 1 {
			return
		}
		// Prefer Unicode path for non-ASCII (CJK etc.)
		if r[0] > 127 {
			if down {
				c.sendUnicode(r[0])
			}
			return
		}
		v, _, _ := procVkKeyScanW.Call(uintptr(r[0]))
		vk = uint16(v & 0xFF)
		if vk == 0xFF {
			return
		}
	}
	scan, _, _ := procMapVirtualKeyW.Call(uintptr(vk), 0)
	var flags uint32
	if !down {
		flags = keyeventfKeyup
	}
	in := winInputKey{Type: inputKeyboard}
	in.Ki.Vk = vk
	in.Ki.Scan = uint16(scan)
	in.Ki.Flags = flags
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func (c *windowsController) sendUnicode(r rune) {
	// KEYEVENTF_UNICODE: Vk=0, Scan=code unit
	down := winInputKey{Type: inputKeyboard}
	down.Ki.Scan = uint16(r)
	down.Ki.Flags = keyeventfUnicode
	up := down
	up.Ki.Flags = keyeventfUnicode | keyeventfKeyup
	procSendInput.Call(1, uintptr(unsafe.Pointer(&down)), unsafe.Sizeof(down))
	procSendInput.Call(1, uintptr(unsafe.Pointer(&up)), unsafe.Sizeof(up))
}

func (c *windowsController) Text(s string) {
	if s == "" {
		return
	}
	for _, r := range s {
		// Surrogate pair for astral plane
		if r > 0xFFFF {
			r1 := 0xD800 + (r-0x10000)>>10
			r2 := 0xDC00 + (r-0x10000)&0x3FF
			c.sendUnicode(rune(r1))
			c.sendUnicode(rune(r2))
			continue
		}
		if r == '\n' || r == '\r' {
			// Enter
			in := winInputKey{Type: inputKeyboard}
			in.Ki.Vk = 0x0D
			procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
			in.Ki.Flags = keyeventfKeyup
			procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
			continue
		}
		if r == '\t' {
			c.Key("Tab", true)
			c.Key("Tab", false)
			continue
		}
		c.sendUnicode(r)
	}
}
