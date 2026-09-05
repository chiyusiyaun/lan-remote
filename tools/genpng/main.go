//go:build ignore

package main

// Writes packaging/linux/icon-client.png and icon-server.png (256px).
import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	// reuse drawing by duplicating minimal logic here for a one-shot tool
)

func main() {
	// minimal: import generated ICO is hard; draw simple 256px via same style
	// We call into a tiny duplicated painter to keep this file standalone.
	writePNG(filepath.Join("packaging", "linux", "icon-client.png"), drawClient(256))
	writePNG(filepath.Join("packaging", "linux", "icon-server.png"), drawServer(256))
	println("png icons written")
}

func writePNG(path string, img image.Image) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func blank(size int) *image.NRGBA {
	return image.NewNRGBA(image.Rect(0, 0, size, size))
}

func fill(img *image.NRGBA, x0, y0, x1, y1 int, c color.NRGBA) {
	b := img.Bounds()
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 >= b.Dx() {
		x1 = b.Dx() - 1
	}
	if y1 >= b.Dy() {
		y1 = b.Dy() - 1
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

func px(img *image.NRGBA, x, y int, c color.NRGBA) {
	b := img.Bounds()
	if x >= 0 && y >= 0 && x < b.Dx() && y < b.Dy() {
		img.SetNRGBA(x, y, c)
	}
}

func drawClient(size int) *image.NRGBA {
	img := blank(size)
	s := float64(size) / 16.0
	bezel := color.NRGBA{0x2B, 0x33, 0x45, 255}
	screen := color.NRGBA{0x3D, 0xC8, 0xFF, 255}
	screen2 := color.NRGBA{0x4F, 0x8C, 0xFF, 255}
	white := color.NRGBA{255, 255, 255, 255}
	stand := color.NRGBA{0x5A, 0x64, 0x78, 255}
	fill(img, int(1*s), int(2*s), int(14*s)-1, int(10*s)-1, bezel)
	sx0, sy0 := int(3*s), int(4*s)
	sx1, sy1 := int(12*s)-1, int(8*s)-1
	mid := (sy0 + sy1) / 2
	for y := sy0; y <= sy1; y++ {
		for x := sx0; x <= sx1; x++ {
			if y <= mid {
				px(img, x, y, screen)
			} else {
				px(img, x, y, screen2)
			}
		}
	}
	chev := func(cx int) {
		n := int(3 * s)
		if n < 2 {
			n = 2
		}
		for i := 0; i < n; i++ {
			for k := -1; k <= 1; k++ {
				px(img, cx+i, mid+k-i/2, white)
			}
		}
	}
	chev(int(5 * s))
	chev(int(9 * s))
	fill(img, int(6*s), int(11*s), int(9*s)-1, int(11*s), stand)
	fill(img, int(5*s), int(12*s), int(10*s)-1, int(12*s), stand)
	return img
}

func drawServer(size int) *image.NRGBA {
	img := blank(size)
	s := float64(size) / 16.0
	body := color.NRGBA{0x2B, 0x33, 0x45, 255}
	slot := color.NRGBA{0x1A, 0x1F, 0x2B, 255}
	accent := color.NRGBA{0x3D, 0xC8, 0xFF, 255}
	ok := color.NRGBA{0x3D, 0xD6, 0x8C, 255}
	edge := color.NRGBA{0x5A, 0x64, 0x78, 255}
	fill(img, int(4*s), int(1*s), int(11*s)-1, int(13*s)-1, body)
	for i := 0; i < 3; i++ {
		y0 := int(float64(3+i*3) * s)
		fill(img, int(5*s), y0, int(10*s)-1, y0+int(1.2*s), slot)
		fill(img, int(5*s), y0+int(0.4*s), int(7*s), y0+int(0.8*s), accent)
	}
	fill(img, int(9*s), int(2*s), int(10*s)-1, int(2*s)+int(1*s), ok)
	fill(img, int(1*s), int(6*s), int(3*s), int(7*s), edge)
	fill(img, int(12*s), int(6*s), int(14*s), int(7*s), edge)
	fill(img, int(2*s), int(6*s), int(2*s)+int(1*s), int(6*s)+int(1*s), accent)
	fill(img, int(13*s), int(6*s), int(13*s)+int(1*s), int(6*s)+int(1*s), accent)
	fill(img, int(3*s), int(13*s), int(12*s)-1, int(14*s), edge)
	return img
}
