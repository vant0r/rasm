package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Crop boundaries - removes status bar, navigation, borders and frame
	// Only the actual picture content remains
	// Original screenshots are 1080x2460
	// Blue frame: cols 33-35 (left), cols 1044-1046 (right)
	// Image content: rows 142-1135, cols 36-1043
	cropTop    = 142
	cropBottom = 1136 // exclusive
	cropLeft   = 36
	cropRight  = 1044 // exclusive

	// Source directory
	sourceDir = "/projects/sandbox/rasm"
)

func main() {
	fmt.Println("=== Bilet Image Processor (Crop Only) ===")
	fmt.Printf("Crop: rows %d-%d, cols %d-%d\n", cropTop, cropBottom-1, cropLeft, cropRight-1)
	fmt.Printf("Result size: %dx%d\n", cropRight-cropLeft, cropBottom-cropTop)
	fmt.Println()

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

		jpegCount := countJPEGs(files)
		fmt.Printf("--- %s (%d files) ---\n", entry.Name(), jpegCount)

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := strings.ToLower(file.Name())
			if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") {
				continue
			}

			imgPath := filepath.Join(biletDir, file.Name())
			if err := processImage(imgPath); err != nil {
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

func processImage(inputPath string) error {
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

	// Crop to just the picture portion
	croppedWidth := cropRight - cropLeft
	croppedHeight := cropBottom - cropTop

	// Create new image with cropped dimensions
	dst := image.NewRGBA(image.Rect(0, 0, croppedWidth, croppedHeight))

	// Draw cropped region
	draw.Draw(dst, dst.Bounds(), srcImg, image.Point{cropLeft, cropTop}, draw.Src)

	// Save output (overwrite original)
	outFile, err := os.Create(inputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	return jpeg.Encode(outFile, dst, &jpeg.Options{Quality: 95})
}
