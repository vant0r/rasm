package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Crop boundaries for the illustration area only (no text, no borders)
const (
	cropTop    = 414
	cropBottom = 1133
	cropLeft   = 36
	cropRight  = 1039
)

func main() {
	baseDir := filepath.Dir(os.Args[0])
	if len(os.Args) > 1 {
		baseDir = os.Args[1]
	}

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
			err := processImage(imgPath)
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

func processImage(imgPath string) error {
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

	// Crop the illustration area only (no text, no logo)
	cropWidth := cropRight - cropLeft
	cropHeight := cropBottom - cropTop

	cropped := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	draw.Draw(cropped, cropped.Bounds(), img, image.Point{X: cropLeft, Y: cropTop}, draw.Src)

	// Save back as JPEG
	out, err := os.Create(imgPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, cropped, &jpeg.Options{Quality: 95})
}
