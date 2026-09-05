package inject

// Controller injects mouse/keyboard events into the local desktop.
type Controller interface {
	MouseMove(x, y int)
	MouseButton(btn string, down bool) // left | right | middle
	MouseScroll(dx, dy int)
	Key(key string, down bool)
	// Text types a Unicode string (supports CJK via KEYEVENTF_UNICODE / xdotool type).
	Text(s string)
}

// New returns a platform-specific Controller.
func New() Controller {
	return newController()
}

