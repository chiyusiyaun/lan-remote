//go:build windows

package capture

import (
	"fmt"
	"image"
	"image/draw"
	"syscall"
	"unsafe"
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	gdi32               = syscall.NewLazyDLL("gdi32.dll")
	getSystemMetrics    = user32.NewProc("GetSystemMetrics")
	getDC               = user32.NewProc("GetDC")
	releaseDC           = user32.NewProc("ReleaseDC")
	createCompatibleDC  = gdi32.NewProc("CreateCompatibleDC")
	createCompatibleBMP = gdi32.NewProc("CreateCompatibleBitmap")
	selectObject        = gdi32.NewProc("SelectObject")
	bitBlt              = gdi32.NewProc("BitBlt")
	getDIBits           = gdi32.NewProc("GetDIBits")
	deleteObject        = gdi32.NewProc("DeleteObject")
	deleteDC            = gdi32.NewProc("DeleteDC")
	setProcessDPIAware  = user32.NewProc("SetProcessDPIAware")
)

const (
	smCXScreen       = 0
	smCYScreen       = 1
	srccopy          = 0x00CC0020
	dibRGBColors     = 0
	biRGB            = 0
	pf32bppARGBShift = 0
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [3]uint32
}

type windowsCapturer struct {
	w, h int
}

func newCapturer() Capturer {
	// Physical pixels, not DPI-virtualized
	setProcessDPIAware.Call()
	w, _, _ := getSystemMetrics.Call(smCXScreen)
	h, _, _ := getSystemMetrics.Call(smCYScreen)
	return &windowsCapturer{w: int(w), h: int(h)}
}

func (c *windowsCapturer) Size() (int, int) {
	c.refresh()
	return c.w, c.h
}

func (c *windowsCapturer) refresh() {
	w, _, _ := getSystemMetrics.Call(smCXScreen)
	h, _, _ := getSystemMetrics.Call(smCYScreen)
	if int(w) > 0 && int(h) > 0 {
		c.w, c.h = int(w), int(h)
	}
}

func (c *windowsCapturer) Capture() (*image.RGBA, error) {
	c.refresh()
	if c.w <= 0 || c.h <= 0 {
		return nil, fmt.Errorf("invalid screen size %dx%d", c.w, c.h)
	}

	hdc, _, err := getDC.Call(0)
	if hdc == 0 {
		return nil, fmt.Errorf("GetDC: %v", err)
	}
	defer releaseDC.Call(0, hdc)

	memDC, _, err := createCompatibleDC.Call(hdc)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC: %v", err)
	}
	defer deleteDC.Call(memDC)

	bmp, _, err := createCompatibleBMP.Call(hdc, uintptr(c.w), uintptr(c.h))
	if bmp == 0 {
		return nil, fmt.Errorf("CreateCompatibleBitmap: %v", err)
	}
	defer deleteObject.Call(bmp)

	old, _, _ := selectObject.Call(memDC, bmp)
	defer selectObject.Call(memDC, old)

	ret, _, err := bitBlt.Call(memDC, 0, 0, uintptr(c.w), uintptr(c.h), hdc, 0, 0, srccopy)
	if ret == 0 {
		return nil, fmt.Errorf("BitBlt: %v", err)
	}

	var bi bitmapInfo
	bi.Header.Size = uint32(unsafe.Sizeof(bi.Header))
	bi.Header.Width = int32(c.w)
	bi.Header.Height = -int32(c.h) // top-down
	bi.Header.Planes = 1
	bi.Header.BitCount = 32
	bi.Header.Compression = biRGB

	buf := make([]byte, c.w*c.h*4)
	ret, _, err = getDIBits.Call(memDC, bmp, 0, uintptr(c.h), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bi)), dibRGBColors)
	if ret == 0 {
		return nil, fmt.Errorf("GetDIBits: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, c.w, c.h))
	// Windows DIB is BGRA; swap to RGBA
	for i := 0; i < len(buf)-3; i += 4 {
		img.Pix[i] = buf[i+2]   // R
		img.Pix[i+1] = buf[i+1] // G
		img.Pix[i+2] = buf[i]   // B
		img.Pix[i+3] = 255      // A
	}
	_ = draw.Src
	return img, nil
}
