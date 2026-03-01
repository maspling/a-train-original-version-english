package graphics

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	EMSGNumEntries = 22
	EMSGTableBytes = EMSGNumEntries * 4
)

func PackEmsg(gameDir, outDir string) error {
	log.SetPrefix("[Packing Emsg] ")
	emsgDir := filepath.Join(gameDir, "edit", "emsg")
	if _, err := os.Stat(filepath.Join(emsgDir, "table.bin")); err == nil {
		fmt.Println("Packing EMSGDAT.PAC...")
		packed, err := packEMSGDAT(emsgDir)
		if err != nil {
			return err
		} else {
			outPath := filepath.Join(outDir, "EMSGDAT.PAC")
			if err := os.WriteFile(outPath, packed, 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

func strideFor(width int) int {
	s := (width + 7) / 8
	if s%2 != 0 {
		s++
	}
	return s
}

func pixelsToPlane(pixels []uint8, width, height, stride int) []byte {
	plane := make([]byte, stride*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if pixels[y*width+x] != 0 {
				byteOff := y*stride + (x >> 3)
				plane[byteOff] |= 1 << uint(7-(x&7))
			}
		}
	}
	return plane
}

func packEMSGDAT(inputDir string) ([]byte, error) {
	tableData, err := os.ReadFile(filepath.Join(inputDir, "table.bin"))
	if err != nil {
		return nil, fmt.Errorf("read table.bin: %w", err)
	}
	if len(tableData) != EMSGTableBytes {
		return nil, fmt.Errorf("table.bin must be exactly %d bytes, got %d", EMSGTableBytes, len(tableData))
	}

	origSizes := make([]uint32, EMSGNumEntries)
	for i := 0; i < EMSGNumEntries; i++ {
		origSizes[i] = binary.LittleEndian.Uint32(tableData[i*4:])
	}

	newSizes := make([]uint32, EMSGNumEntries)
	bodyParts := make([][]byte, EMSGNumEntries)

	for i := 0; i < EMSGNumEntries; i++ {
		stem := fmt.Sprintf("EMSG%02d", i)
		hdrPath := filepath.Join(inputDir, stem+".hdr")
		pngPath := filepath.Join(inputDir, stem+".png")

		hdrOrig, err := os.ReadFile(hdrPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", hdrPath, err)
		}
		hdr := make([]byte, len(hdrOrig))
		copy(hdr, hdrOrig)

		origW := binary.LittleEndian.Uint16(hdr[0x1c:])
		origH := binary.LittleEndian.Uint16(hdr[0x1a:])
		origSW := binary.LittleEndian.Uint16(hdr[0x18:])
		origS := int(origSW) * 2

		w, h, pixels, err := ReadIndexedPNG(pngPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", pngPath, err)
		}

		stride := origS
		if w != int(origW) || h != int(origH) {
			stride = strideFor(w)
			binary.LittleEndian.PutUint16(hdr[0x18:], uint16(stride/2))
			binary.LittleEndian.PutUint16(hdr[0x1a:], uint16(h))
			binary.LittleEndian.PutUint16(hdr[0x1c:], uint16(w))
		}

		plane := pixelsToPlane(pixels, w, h, stride)
		entryData := make([]byte, 0, len(hdr)+len(plane))
		entryData = append(entryData, hdr...)
		entryData = append(entryData, plane...)
		newSize := uint32(len(entryData))

		if newSize < origSizes[i] {
			pad := make([]byte, origSizes[i]-newSize)
			entryData = append(entryData, pad...)
			newSize = origSizes[i]
		} else if newSize > origSizes[i] {
			fmt.Printf("  WARNING: %s entry is %d B, larger than original slot %d B\n",
				stem, newSize, origSizes[i])
		}

		newSizes[i] = newSize
		bodyParts[i] = entryData
		fmt.Printf("  %s  %dx%d  %d B\n", stem, w, h, newSize)
	}

	newTable := make([]byte, EMSGTableBytes)
	for i := 0; i < EMSGNumEntries; i++ {
		binary.LittleEndian.PutUint32(newTable[i*4:], newSizes[i])
	}

	totalBody := 0
	for _, part := range bodyParts {
		totalBody += len(part)
	}

	result := make([]byte, 0, EMSGTableBytes+totalBody)
	result = append(result, newTable...)
	for _, part := range bodyParts {
		result = append(result, part...)
	}
	return result, nil
}
