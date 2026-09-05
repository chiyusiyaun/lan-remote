package capture

import "image"

// Capturer grabs a full screenshot of the primary display.
type Capturer interface {
	Capture() (*image.RGBA, error)
	Size() (w, h int)
}

// New returns a platform-specific Capturer.
func New() Capturer {
	return newCapturer()
}
