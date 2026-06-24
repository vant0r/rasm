package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Crop boundaries for the illustration area (consistent across all 1080x2460 screenshots)
const (
	cropTop    = 414
	cropBottom = 1133
	cropLeft   = 36
	cropRight  = 1039
)

// Logo settings
const (
	logoSizeRatio  = 0.18  // 18% of cropped image height
	logoMarginRight  = 10
	logoMarginBottom = 10
)

func main() {
	baseDir := filepath.Dir(os.Args[0])
	if len(os.Args) > 1 {
		baseDir = os.Args[1]
	}

	// Load logo
	logoPath := filepath.Join(baseDir, "logo.png")
	logo, err := loadPNG(logoPath)
	if err != nil {
		fmt.Printf("Error loading logo from %s: %v\n", logoPath, err)
		os.Exit(1)
	}
	fmt.Printf("Logo loaded: %dx%d\n", logo.Bounds().Dx(), logo.Bounds().Dy())

	folders := []string{"Bilet 1", "Bilet 2", "Bilet 3", "Bilet 4", "Bilet 5"}
	totalProcessed := 0

	for _, folder := range folders {
		folderPath := filepath.Join(baseDir, "..", folder)
		fmt.Printf("\nProcessing %s...\n", folder)

		entries, err := os.ReadDir(folderPath)
		if err != nil {
			fmt.Printf("  Error reading folder: %v\n", err)
			continue
		}

		// Sort entries by name
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".jpg") && !strings.HasSuffix(lower, ".jpeg") {
				continue
			}

			imgPath := filepath.Join(folderPath, name)
			err := processImage(imgPath, logo)
			if err != nil {
				fmt.Printf("  ERROR %s: %v\n", name, err)
			} else {
				fmt.Printf("  Processed: %s\n", name)
				totalProcessed++
			}
		}
	}

	fmt.Printf("\n%s\nDone! Total images processed: %d\n", strings.Repeat("=", 50), totalProcessed)
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func processImage(imgPath string, logo image.Image) error {
	// Open JPEG
	f, err := os.Open(imgPath)
	if err != nil {
		return err
	}

	img, err := jpeg.Decode(f)
	f.Close()
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Verify expected dimensions
	if width != 1080 || height != 2460 {
		return fmt.Errorf("unexpected size %dx%d (expected 1080x2460)", width, height)
	}

	// Crop the illustration area
	cropWidth := cropRight - cropLeft
	cropHeight := cropBottom - cropTop

	cropped := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	draw.Draw(cropped, cropped.Bounds(), img, image.Point{X: cropLeft, Y: cropTop}, draw.Src)

	// Resize logo
	logoSize := int(float64(cropHeight) * logoSizeRatio)
	resizedLogo := resizeImage(logo, logoSize, logoSize)

	// Calculate logo position (bottom-right corner)
	logoX := cropWidth - logoSize - logoMarginRight
	logoY := cropHeight - logoSize - logoMarginBottom

	// Draw logo with alpha compositing
	logoRect := image.Rect(logoX, logoY, logoX+logoSize, logoY+logoSize)
	draw.Draw(cropped, logoRect, resizedLogo, image.Point{}, draw.Over)

	// Save back as JPEG
	out, err := os.Create(imgPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, cropped, &jpeg.Options{Quality: 95})
}

// resizeImage resizes an image to the given dimensions using bilinear interpolation
func resizeImage(src image.Image, newWidth, newHeight int) image.Image {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	xRatio := float64(srcW) / float64(newWidth)
	yRatio := float64(srcH) / float64(newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			// Source coordinates
			srcX := float64(x) * xRatio
			srcY := float64(y) * yRatio

			// Bilinear interpolation
			x0 := int(math.Floor(srcX))
			y0 := int(math.Floor(srcY))
			x1 := x0 + 1
			y1 := y0 + 1

			if x1 >= srcW {
				x1 = srcW - 1
			}
			if y1 >= srcH {
				y1 = srcH - 1
			}

			// Get the four surrounding pixels
			r00, g00, b00, a00 := src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y0).RGBA()
			r10, g10, b10, a10 := src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y0).RGBA()
			r01, g01, b01, a01 := src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y1).RGBA()
			r11, g11, b11, a11 := src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y1).RGBA()

			// Interpolation weights
			xFrac := srcX - float64(x0)
			yFrac := srcY - float64(y0)

			// Bilinear interpolation for each channel
			r := bilinear(r00, r10, r01, r11, xFrac, yFrac)
			g := bilinear(g00, g10, g01, g11, xFrac, yFrac)
			b := bilinear(b00, b10, b01, b11, xFrac, yFrac)
			a := bilinear(a00, a10, a01, a11, xFrac, yFrac)

			dst.Set(x, y, color.RGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

func bilinear(v00, v10, v01, v11 uint32, xFrac, yFrac float64) uint32 {
	top := float64(v00)*(1-xFrac) + float64(v10)*xFrac
	bottom := float64(v01)*(1-xFrac) + float64(v11)*xFrac
	return uint32(top*(1-yFrac) + bottom*yFrac)
}
