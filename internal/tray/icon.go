package tray

// Icon: 16×16 remote-desktop mark — dark bezel + cyan screen + link arrows.
var iconBytes = buildICO()

func buildICO() []byte {
	const s = 16
	px := make([]byte, s*s*4)
	set := func(x, y int, r, g, b, a byte) {
		if x < 0 || y < 0 || x >= s || y >= s {
			return
		}
		i := (y*s + x) * 4
		px[i] = b
		px[i+1] = g
		px[i+2] = r
		px[i+3] = a
	}

	// colors
	bezel := [3]byte{0x2B, 0x33, 0x45} // dark slate
	screen := [3]byte{0x3D, 0xC8, 0xFF} // cyan
	screen2 := [3]byte{0x4F, 0x8C, 0xFF} // blue accent
	arrow := [3]byte{0xFF, 0xFF, 0xFF}
	stand := [3]byte{0x5A, 0x64, 0x78}

	// monitor body (1,2)-(14,10)
	for y := 2; y <= 10; y++ {
		for x := 1; x <= 14; x++ {
			// rounded corners skip
			if (y == 2 || y == 10) && (x == 1 || x == 14) {
				continue
			}
			if (y == 2 || y == 10) && (x == 2 || x == 13) {
				set(x, y, bezel[0], bezel[1], bezel[2], 255)
				continue
			}
			set(x, y, bezel[0], bezel[1], bezel[2], 255)
		}
	}
	// screen inner (3,4)-(12,8)
	for y := 4; y <= 8; y++ {
		for x := 3; x <= 12; x++ {
			// gradient-ish cyan
			if y <= 5 {
				set(x, y, screen[0], screen[1], screen[2], 255)
			} else {
				set(x, y, screen2[0], screen2[1], screen2[2], 255)
			}
		}
	}
	// link arrows on screen (two chevrons pointing right)
	// left chevron
	for i := 0; i < 4; i++ {
		set(5+i, 6-i, arrow[0], arrow[1], arrow[2], 255)
		set(5+i, 7-i, arrow[0], arrow[1], arrow[2], 255)
		set(5+i, 8-i, arrow[0], arrow[1], arrow[2], 255)
	}
	// right chevron
	for i := 0; i < 4; i++ {
		set(9+i, 6-i, arrow[0], arrow[1], arrow[2], 255)
		set(9+i, 7-i, arrow[0], arrow[1], arrow[2], 255)
		set(9+i, 8-i, arrow[0], arrow[1], arrow[2], 255)
	}
	// stand
	for x := 6; x <= 9; x++ {
		set(x, 11, stand[0], stand[1], stand[2], 255)
	}
	for x := 5; x <= 10; x++ {
		set(x, 12, stand[0], stand[1], stand[2], 255)
	}

	andMask := make([]byte, s*s/8)
	bih := make([]byte, 40)
	putU32(bih[0:], 40)
	putI32(bih[4:], int32(s))
	putI32(bih[8:], int32(s*2))
	putU16(bih[12:], 1)
	putU16(bih[14:], 32)

	flipped := make([]byte, len(px))
	stride := s * 4
	for y := 0; y < s; y++ {
		copy(flipped[y*stride:], px[(s-1-y)*stride:])
	}
	img := append(bih, flipped...)
	img = append(img, andMask...)

	ico := make([]byte, 0, 22+len(img))
	ico = append(ico, 0, 0, 1, 0, 1, 0)
	entry := make([]byte, 16)
	entry[0] = byte(s)
	entry[1] = byte(s)
	putU16(entry[4:], 1)
	putU16(entry[6:], 32)
	putU32(entry[8:], uint32(len(img)))
	putU32(entry[12:], 22)
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
