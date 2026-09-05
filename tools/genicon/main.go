//go:build ignore

package main

// Generates a multi-size ICO for LAN Remote.
import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
)

func drawIcon(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := float64(size) / 16.0
	px := func(x, y int, c color.NRGBA) {
		if x >= 0 && y >= 0 && x < size && y < size {
			img.SetNRGBA(x, y, c)
		}
	}
	fillRect := func(x0, y0, x1, y1 int, c color.NRGBA) {
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				px(x, y, c)
			}
		}
	}
	bezel := color.NRGBA{0x2B, 0x33, 0x45, 255}
	screen := color.NRGBA{0x3D, 0xC8, 0xFF, 255}
	screen2 := color.NRGBA{0x4F, 0x8C, 0xFF, 255}
	white := color.NRGBA{255, 255, 255, 255}
	stand := color.NRGBA{0x5A, 0x64, 0x78, 255}
	transparent := color.NRGBA{0, 0, 0, 0}

	fillRect(0, 0, size-1, size-1, transparent)

	x0, y0 := int(1*scale), int(2*scale)
	x1, y1 := int(14*scale)-1, int(10*scale)-1
	if x1 < x0 {
		x1 = x0
	}
	if y1 < y0 {
		y1 = y0
	}
	fillRect(x0, y0, x1, y1, bezel)

	sx0, sy0 := int(3*scale), int(4*scale)
	sx1, sy1 := int(12*scale)-1, int(8*scale)-1
	if sx1 < sx0 {
		sx1 = sx0
	}
	if sy1 < sy0 {
		sy1 = sy0
	}
	mid := (sy0 + sy1) / 2
	for y := sy0; y <= sy1; y++ {
		for x := sx0; x <= sx1; x++ {
			if y <= mid {
				px(x, y, screen)
			} else {
				px(x, y, screen2)
			}
		}
	}

	// chevrons
	chev := func(cx int) {
		n := int(3 * scale)
		if n < 2 {
			n = 2
		}
		for i := 0; i < n; i++ {
			for k := -1; k <= 1; k++ {
				px(cx+i, mid+k-i/2, white)
			}
		}
	}
	chev(int(5 * scale))
	chev(int(9 * scale))

	fillRect(int(6*scale), int(11*scale), int(9*scale)-1, int(11*scale), stand)
	fillRect(int(5*scale), int(12*scale), int(10*scale)-1, int(12*scale), stand)
	return img
}

func encodePNG(im image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, im)
	return buf.Bytes()
}

func encodeICO(images []image.Image) []byte {
	type ent struct {
		w, h byte
		data []byte
	}
	var ents []ent
	for _, im := range images {
		b := im.Bounds()
		w, h := b.Dx(), b.Dy()
		ew, eh := byte(w), byte(h)
		if w >= 256 {
			ew = 0
		}
		if h >= 256 {
			eh = 0
		}
		ents = append(ents, ent{ew, eh, encodePNG(im)})
	}
	var out []byte
	out = append(out, 0, 0, 1, 0, byte(len(ents)), 0)
	offset := 6 + 16*len(ents)
	for _, e := range ents {
		row := make([]byte, 16)
		row[0] = e.w
		row[1] = e.h
		binary.LittleEndian.PutUint16(row[4:], 1)
		binary.LittleEndian.PutUint16(row[6:], 32)
		binary.LittleEndian.PutUint32(row[8:], uint32(len(e.data)))
		binary.LittleEndian.PutUint32(row[12:], uint32(offset))
		out = append(out, row...)
		offset += len(e.data)
	}
	for _, e := range ents {
		out = append(out, e.data...)
	}
	return out
}

func main() {
	var imgs []image.Image
	for _, s := range []int{16, 32, 48, 64, 128, 256} {
		imgs = append(imgs, drawIcon(s))
	}
	data := encodeICO(imgs)
	if err := os.WriteFile("icon.ico", data, 0o644); err != nil {
		panic(err)
	}
	println("wrote icon.ico", len(data), "bytes")
}
