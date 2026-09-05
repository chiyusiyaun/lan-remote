//go:build linux

package capture

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Linux capturer prefers scrot/import (ImageMagick) which work under X11.
// Wayland compositors generally block global capture; document that limitation.
type linuxCapturer struct{}

func newCapturer() Capturer { return &linuxCapturer{} }

func (c *linuxCapturer) Size() (int, int) {
	// Best-effort from xdpyinfo or xrandr
	out, err := exec.Command("xdpyinfo").Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "dimensions:") {
				// dimensions:    1920x1080 pixels (508x285 millimeters)
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					parts := strings.Split(fields[1], "x")
					if len(parts) == 2 {
						w, _ := strconv.Atoi(parts[0])
						h, _ := strconv.Atoi(parts[1])
						if w > 0 && h > 0 {
							return w, h
						}
					}
				}
			}
		}
	}
	return 1920, 1080
}

func (c *linuxCapturer) Capture() (*image.RGBA, error) {
	tmp := os.TempDir() + "/lan_remote_cap.png"
	// scrot is common on X11; fall back to import (ImageMagick)
	var err error
	if _, e := exec.LookPath("scrot"); e == nil {
		err = exec.Command("scrot", "-o", tmp).Run()
	} else if _, e := exec.LookPath("import"); e == nil {
		err = exec.Command("import", "-window", "root", tmp).Run()
	} else {
		return nil, fmt.Errorf("need scrot or imagemagick (import) on PATH for Linux capture")
	}
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)

	f, err := os.Open(tmp)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	rgba := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			rgba.Set(x, y, img.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return rgba, nil
}
