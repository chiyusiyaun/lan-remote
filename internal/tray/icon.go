package tray

// Minimal 16x16 ICO (blue rounded square) for the tray icon.
var iconBytes = buildICO()

func buildICO() []byte {
	// 16x16 BGRA pixels
	const s = 16
	px := make([]byte, s*s*4)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			i := (y*s + x) * 4
			// simple filled circle-ish
			dx, dy := x-8, y-8
			if dx*dx+dy*dy < 36 {
				px[i] = 0xE0 // B
				px[i+1] = 0x8C // G
				px[i+2] = 0x4F // R
				px[i+3] = 0xFF
			} else {
				px[i+3] = 0
			}
		}
	}
	// AND mask: all 0 (alpha used)
	andMask := make([]byte, s*s/8)

	// BITMAPINFOHEADER 40 bytes
	bih := make([]byte, 40)
	putU32(bih[0:], 40)
	putI32(bih[4:], int32(s))
	putI32(bih[8:], int32(s*2))
	putU16(bih[12:], 1)
	putU16(bih[14:], 32)
	// rest zero

	xor := make([]byte, len(px))
	copy(xor, px)
	// flip bottom-up for DIB
	flipped := make([]byte, len(xor))
	stride := s * 4
	for y := 0; y < s; y++ {
		copy(flipped[y*stride:], xor[(s-1-y)*stride:])
	}

	img := append(bih, flipped...)
	img = append(img, andMask...)

	// ICONDIR
	ico := make([]byte, 0, 6+16+len(img))
	ico = append(ico, 0, 0, 1, 0, 1, 0) // reserved, type=1, count=1
	// ICONDIRENTRY
	entry := make([]byte, 16)
	entry[0] = byte(s)
	entry[1] = byte(s)
	entry[2] = 0
	entry[3] = 0
	putU16(entry[4:], 1)  // planes
	putU16(entry[6:], 32) // bitcount
	putU32(entry[8:], uint32(len(img)))
	putU32(entry[12:], 22) // offset
	ico = append(ico, entry...)
	ico = append(ico, img...)
	return ico
}

func putU16(b []byte, v uint16) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
}
func putU32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}
func putI32(b []byte, v int32) { putU32(b, uint32(v)) }
