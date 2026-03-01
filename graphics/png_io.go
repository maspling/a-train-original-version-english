package graphics

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

func ReadIndexedPNG(path string) (int, int, []uint8, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("decode PNG %s: %w", path, err)
	}

	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if pImg, ok := img.(*image.Paletted); ok {
		pixels := make([]uint8, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				pixels[y*w+x] = pImg.ColorIndexAt(bounds.Min.X+x, bounds.Min.Y+y)
			}
		}
		return w, h, pixels, nil
	}

	pixels := make([]uint8, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			lum := (r + g + b) / 3 >> 8
			if lum > 128 {
				pixels[y*w+x] = 0
			} else {
				pixels[y*w+x] = 1
			}
		}
	}
	return w, h, pixels, nil
}

func WriteIndexedPNG(path string, w, h int, pixels []uint8, palette [][3]uint8) error {
	pal := make(color.Palette, len(palette))
	for i, c := range palette {
		pal[i] = color.RGBA{R: c[0], G: c[1], B: c[2], A: 255}
	}

	img := image.NewPaletted(image.Rect(0, 0, w, h), pal)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetColorIndex(x, y, pixels[y*w+x])
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
