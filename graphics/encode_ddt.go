package graphics

import (
	"encoding/binary"
	"fmt"
	"os"
)

func EncodeDDT(origDD9, editedPNG string) ([]byte, error) {
	origData, err := os.ReadFile(origDD9)
	if err != nil {
		return nil, fmt.Errorf("read original DD9: %w", err)
	}

	dec := DDTDecompress(origData)
	if len(dec) < 0x20 {
		return nil, fmt.Errorf("decompressed data too short for header")
	}

	header := make([]byte, 0x20)
	copy(header, dec[:0x20])

	extType := header[9]
	numPlanes := 4
	switch extType {
	case 'D':
		numPlanes = 4
	case 'C':
		numPlanes = 3
	case 'B':
		numPlanes = 2
	case 'M':
		numPlanes = 1
	}

	origStrideW := binary.LittleEndian.Uint16(header[0x18:])
	origHeight := binary.LittleEndian.Uint16(header[0x1a:])
	origWidth := binary.LittleEndian.Uint16(header[0x1c:])

	w, h, flatPixels, err := ReadIndexedPNG(editedPNG)
	if err != nil {
		return nil, fmt.Errorf("read PNG: %w", err)
	}

	var stride int
	if w != int(origWidth) || h != int(origHeight) {
		stride = (w + 7) / 8
		if stride%2 != 0 {
			stride++
		}
		binary.LittleEndian.PutUint16(header[0x18:], uint16(stride/2))
		binary.LittleEndian.PutUint16(header[0x1a:], uint16(h))
		binary.LittleEndian.PutUint16(header[0x1c:], uint16(w))
	} else {
		stride = int(origStrideW) * 2
	}

	planeSize := stride * h
	planes := make([][]byte, numPlanes)
	for p := range planes {
		planes[p] = make([]byte, planeSize)
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := flatPixels[y*w+x] & 0x0F
			byteOff := y*stride + (x >> 3)
			bitPos := uint(7 - (x & 7))
			for p := 0; p < numPlanes; p++ {
				if (c>>uint(p))&1 != 0 {
					planes[p][byteOff] |= 1 << bitPos
				}
			}
		}
	}

	uncompressed := make([]byte, 0, 0x20+numPlanes*planeSize)
	uncompressed = append(uncompressed, header...)
	for _, plane := range planes {
		uncompressed = append(uncompressed, plane...)
	}

	compressed := DDTCompress(uncompressed)

	return compressed, nil
}
