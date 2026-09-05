//go:build linux

package inject

import (
	"os/exec"
	"strconv"
	"strings"
)

// Linux controller uses xdotool (X11). Wayland requires a different path.
type linuxController struct{}

func newController() Controller { return &linuxController{} }

func (c *linuxController) MouseMove(x, y int) {
	exec.Command("xdotool", "mousemove", strconv.Itoa(x), strconv.Itoa(y)).Run()
}

func (c *linuxController) MouseButton(btn string, down bool) {
	num := "1"
	switch btn {
	case "middle":
		num = "2"
	case "right":
		num = "3"
	}
	action := "mousedown"
	if !down {
		action = "mouseup"
	}
	exec.Command("xdotool", action, num).Run()
}

func (c *linuxController) MouseScroll(dx, dy int) {
	// xdotool click 4/5 = up/down
	if dy < 0 {
		exec.Command("xdotool", "click", "5").Run()
	} else if dy > 0 {
		exec.Command("xdotool", "click", "4").Run()
	}
	if dx < 0 {
		exec.Command("xdotool", "click", "6").Run()
	} else if dx > 0 {
		exec.Command("xdotool", "click", "7").Run()
	}
}

func (c *linuxController) Key(key string, down bool) {
	name := key
	// Normalize a few names
	name = strings.ReplaceAll(name, " ", "")
	if key == " " {
		name = "space"
	}
	cmd := "keydown"
	if !down {
		cmd = "keyup"
	}
	exec.Command("xdotool", cmd, name).Run()
}

func (c *linuxController) Text(s string) {
	if s == "" {
		return
	}
	// --clearmodifiers avoids stuck modifier state; delay 12ms per char
	exec.Command("xdotool", "type", "--clearmodifiers", "--delay", "12", "--", s).Run()
}
