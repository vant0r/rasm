package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Crop boundaries determined by pixel variance analysis
	// Original screenshots are 1080x2460
	// Image region: rows 93 to 1129 (inclusive), full width 1080
	cropTop    = 93
	cropBottom = 1130 // exclusive
	cropLeft   = 0
	cropRight  = 1080

	// Logo placement - bottom right corner to cover Telegram watermark
	// Logo size as ratio of cropped image width
	logoSizeRatio   = 0.15
	logoMarginRight = 10
	logoMarginBottom = 10

	// Source directory
	sourceDir = "/projects/sandbox/rasm"
	logoPath  = "/projects/sandbox/rasm/processor/logo.png"
)

func main() {
	fmt.Println("=== Vatanparvar Bilet Image Processor ===")
	fmt.Printf("Crop: top=%d, bottom=%d (rows), full width\n", cropTop, cropBottom)
	fmt.Printf("Cropped size: %dx%d\n", cropRight-cropLeft, cropBottom-cropTop)
	fmt.Println()

	// Load logo
	logo, err := loadLogo()
	if err != nil {
		fmt.Printf("Error loading logo: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Logo loaded: %dx%d\n\n", logo.Bounds().Dx(), logo.Bounds().Dy())

	// Find all Bilet folders
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	processed := 0
	errCount := 0

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "Bilet") {
			continue
		}

		biletDir := filepath.Join(sourceDir, entry.Name())
		files, err := os.ReadDir(biletDir)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", biletDir, err)
			continue
		}

		fmt.Printf("--- %s (%d files) ---\n", entry.Name(), countJPEGs(files))

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := strings.ToLower(file.Name())
			if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") {
				continue
			}

			imgPath := filepath.Join(biletDir, file.Name())
			if err := processImage(imgPath, logo); err != nil {
				fmt.Printf("  ERROR: %s - %v\n", file.Name(), err)
				errCount++
			} else {
				fmt.Printf("  OK: %s\n", file.Name())
				processed++
			}
		}
	}

	fmt.Printf("\n=== Done! Processed: %d, Errors: %d ===\n", processed, errCount)
}

func countJPEGs(files []os.DirEntry) int {
	count := 0
	for _, f := range files {
		name := strings.ToLower(f.Name())
		if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") {
			count++
		}
	}
	return count
}

func loadLogo() (image.Image, error) {
	f, err := os.Open(logoPath)
	if err != nil {
		return nil, fmt.Errorf("open logo: %w", err)
	}
	defer f.Close()

	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode logo: %w", err)
	}

	return img, nil
}

func processImage(inputPath string, logo image.Image) error {
	// Open the source image
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	// Decode JPEG
	srcImg, err := jpeg.Decode(f)
	f.Close()
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	// Crop to just the image portion
	croppedWidth := cropRight - cropLeft
	croppedHeight := cropBottom - cropTop

	// Create new image with cropped dimensions
	dst := image.NewRGBA(image.Rect(0, 0, croppedWidth, croppedHeight))

	// Draw cropped region
	draw.Draw(dst, dst.Bounds(), srcImg, image.Point{cropLeft, cropTop}, draw.Src)

	// Calculate logo size
	logoSize := int(float64(croppedWidth) * logoSizeRatio)

	// Resize logo
	resizedLogo := resizeImage(logo, logoSize, logoSize)

	// Position at bottom-right corner (to cover Telegram watermark)
	logoX := croppedWidth - logoSize - logoMarginRight
	logoY := croppedHeight - logoSize - logoMarginBottom
	logoRect := image.Rect(logoX, logoY, logoX+logoSize, logoY+logoSize)

	// Draw logo with alpha blending
	draw.Draw(dst, logoRect, resizedLogo, image.Point{0, 0}, draw.Over)

	// Save output (overwrite original)
	outFile, err := os.Create(inputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	return jpeg.Encode(outFile, dst, &jpeg.Options{Quality: 92})
}

// resizeImage scales an image to the target width/height using bilinear interpolation
func resizeImage(src image.Image, targetW, targetH int) *image.RGBA {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))

	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := float64(x) * float64(srcW) / float64(targetW)
			srcY := float64(y) * float64(srcH) / float64(targetH)

			x0 := int(srcX)
			y0 := int(srcY)
			x1 := x0 + 1
			y1 := y0 + 1

			if x1 >= srcW {
				x1 = srcW - 1
			}
			if y1 >= srcH {
				y1 = srcH - 1
			}

			fx := srcX - float64(x0)
			fy := srcY - float64(y0)

			r00, g00, b00, a00 := src.At(x0+srcBounds.Min.X, y0+srcBounds.Min.Y).RGBA()
			r10, g10, b10, a10 := src.At(x1+srcBounds.Min.X, y0+srcBounds.Min.Y).RGBA()
			r01, g01, b01, a01 := src.At(x0+srcBounds.Min.X, y1+srcBounds.Min.Y).RGBA()
			r11, g11, b11, a11 := src.At(x1+srcBounds.Min.X, y1+srcBounds.Min.Y).RGBA()

			r := lerp2d(r00, r10, r01, r11, fx, fy)
			g := lerp2d(g00, g10, g01, g11, fx, fy)
			b := lerp2d(b00, b10, b01, b11, fx, fy)
			a := lerp2d(a00, a10, a01, a11, fx, fy)

			dst.Set(x, y, color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: uint8(a >> 8),
			})
		}
	}

	return dst
}

func lerp2d(v00, v10, v01, v11 uint32, fx, fy float64) uint32 {
	top := float64(v00)*(1-fx) + float64(v10)*fx
	bottom := float64(v01)*(1-fx) + float64(v11)*fx
	return uint32(top*(1-fy) + bottom*fy)
}
