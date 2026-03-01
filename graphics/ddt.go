package graphics

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

type DDTHeader struct {
	Raw     [0x20]byte
	Name    string
	ExtType byte
	StrideW uint16
	Height  uint16
	Width   uint16
}

func PatchDDT(gameDir, outDir string) error {
	log.SetPrefix("[Patching DDT] ")
	editDir := filepath.Join(gameDir, "edit")
	dd9Files, _ := filepath.Glob(filepath.Join(gameDir, "*.DD9"))
	for _, dd9Path := range dd9Files {
		stem := strings.TrimSuffix(filepath.Base(dd9Path), filepath.Ext(dd9Path))
		pngPath := filepath.Join(editDir, stem+".png")
		if _, err := os.Stat(pngPath); err != nil {
			continue
		}

		log.Printf("Encoding %s...\n", filepath.Base(dd9Path))
		encoded, err := EncodeDDT(dd9Path, pngPath)
		if err != nil {
			return err
		}
		outPath := filepath.Join(outDir, filepath.Base(dd9Path))
		if err := os.WriteFile(outPath, encoded, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (h DDTHeader) NumPlanes() int {
	switch h.ExtType {
	case 'D':
		return 4
	case 'C':
		return 3
	case 'B':
		return 2
	case 'M':
		return 1
	default:
		return 0
	}
}

func (h DDTHeader) Stride() int {
	return int(h.StrideW) * 2
}

func DDTDecompress(data []byte) []byte {
	dst := make([]byte, 0, len(data)*2)
	i := 0
	for i < len(data) {
		b := data[i]
		i++
		if b == 0x99 || b == 0xBB {
			if i >= len(data) {
				break
			}
			dst = append(dst, data[i])
			i++
		} else if b&0xF0 == 0x90 {
			n := int(b & 0x0F)
			for j := 0; j < n; j++ {
				dst = append(dst, 0x00)
			}
		} else if b&0xF0 == 0xB0 {
			n := int(b & 0x0F)
			for j := 0; j < n; j++ {
				dst = append(dst, 0xFF)
			}
		} else {
			dst = append(dst, b)
		}
	}
	return dst
}

func DDTCompress(data []byte) []byte {
	dst := make([]byte, 0, len(data))
	i := 0
	n := len(data)
	for i < n {
		b := data[i]
		if b == 0x00 {

			j := i
			for j < n && data[j] == 0x00 {
				j++
			}
			run := j - i
			i = j
			for run > 0 {
				chunk := run
				if chunk > 15 {
					chunk = 15
				}
				if chunk == 9 {
					chunk = 8
				}
				dst = append(dst, byte(0x90|chunk))
				run -= chunk
			}
		} else if b == 0xFF {

			j := i
			for j < n && data[j] == 0xFF {
				j++
			}
			run := j - i
			i = j
			for run > 0 {
				chunk := run
				if chunk > 15 {
					chunk = 15
				}
				if chunk == 11 {
					chunk = 10
				}
				dst = append(dst, byte(0xB0|chunk))
				run -= chunk
			}
		} else if b&0xF0 == 0x90 || b&0xF0 == 0xB0 {
			dst = append(dst, 0x99, b)
			i++
		} else {
			dst = append(dst, b)
			i++
		}
	}
	return dst
}
