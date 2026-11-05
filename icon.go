package main

import (
	"encoding/binary"
	"unsafe"
)

// makeIcon draws the tray icon (a white four-way arrow on a rounded square of
// the given color) and returns an HICON. Drawing it here keeps the app a single
// self-contained executable.
func makeIcon(size int, r, g, b uint8) uintptr {
	const (
		headerSize = 40
		samples    = 4 // supersampling per axis, for anti-aliasing
	)
	maskStride := (size + 31) / 32 * 4
	pixelBytes := size * size * 4
	buf := make([]byte, headerSize+pixelBytes+maskStride*size)

	// BITMAPINFOHEADER; the height covers the color image plus the AND mask.
	binary.LittleEndian.PutUint32(buf[0:], headerSize)
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
	binary.LittleEndian.PutUint32(buf[8:], uint32(size*2))
	binary.LittleEndian.PutUint16(buf[12:], 1)
	binary.LittleEndian.PutUint16(buf[14:], 32)
	binary.LittleEndian.PutUint32(buf[20:], uint32(len(buf)-headerSize))

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var inside, arrow int
			for sy := 0; sy < samples; sy++ {
				for sx := 0; sx < samples; sx++ {
					u := (float64(x) + (float64(sx)+0.5)/samples) / float64(size)
					v := (float64(y) + (float64(sy)+0.5)/samples) / float64(size)
					if inRoundedSquare(u, v) {
						inside++
						if inArrow(u-0.5, v-0.5) {
							arrow++
						}
					}
				}
			}
			alpha := inside * 255 / (samples * samples)
			blend := func(c uint8) byte {
				if inside == 0 {
					return 0
				}
				return byte((int(c)*(inside-arrow) + 255*arrow) / inside)
			}
			row := size - 1 - y // bottom-up
			p := headerSize + (row*size+x)*4
			buf[p], buf[p+1], buf[p+2], buf[p+3] = blend(b), blend(g), blend(r), byte(alpha)
			if alpha == 0 {
				buf[headerSize+pixelBytes+row*maskStride+x/8] |= 0x80 >> (x % 8)
			}
		}
	}

	h, _, _ := procCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)),
		1, 0x00030000, uintptr(size), uintptr(size), 0)
	return h
}

func inRoundedSquare(u, v float64) bool {
	const lo, hi, radius = 0.03, 0.97, 0.22
	cu := min(max(u, lo+radius), hi-radius)
	cv := min(max(v, lo+radius), hi-radius)
	du, dv := u-cu, v-cv
	return u >= lo && u <= hi && v >= lo && v <= hi && du*du+dv*dv <= radius*radius
}

// inArrow tests a point (relative to the center) against four arrows pointing
// outwards, by rotating it onto the right-pointing arm.
func inArrow(x, y float64) bool {
	for i := 0; i < 4; i++ {
		x, y = y, -x
		ay := y
		if ay < 0 {
			ay = -ay
		}
		shaft := x >= 0 && x <= 0.26 && ay <= 0.055
		head := x >= 0.22 && x <= 0.40 && ay <= 0.14*(0.40-x)/0.18
		if shaft || head {
			return true
		}
	}
	return false
}
