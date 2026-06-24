package main

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

// findIllustrationBounds detects the illustration area in a screenshot
// Returns the crop rectangle (top, bottom, left, right) or zeros if no illustration found
func findIllustrationBounds(img image.Image) (top, bottom, left, right int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Strategy: The illustration is a large colored rectangle (driving scene)
	// We scan from y=450 downward looking for rows where >70% of central pixels are non-white
	consecutiveHigh := 0
	top = 0
	for y := 450; y < height/2; y++ {
		nonWhite := 0
		total := 0
		for x := 80; x < width-80; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			total++
			if r8 < 235 || g8 < 235 || b8 < 235 {
				nonWhite++
			}
		}
		ratio := float64(nonWhite) / float64(total)
		if ratio > 0.70 {
			consecutiveHigh++
			if consecutiveHigh >= 30 {
				top = y - 30 + 1
				break
			}
		} else {
			consecutiveHigh = 0
		}
	}

	if top == 0 {
		return 0, 0, 0, 0
	}

	// Find the bottom of illustration
	bottom = 0
	lowCount := 0
	for y := top + 100; y < height; y++ {
		nonWhite := 0
		total := 0
		for x := 80; x < width-80; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			total++
			if r8 < 235 || g8 < 235 || b8 < 235 {
				nonWhite++
			}
		}
		ratio := float64(nonWhite) / float64(total)
		if ratio < 0.15 {
			lowCount++
			if lowCount > 5 {
				bottom = y - lowCount
				break
			}
		} else {
			lowCount = 0
		}
	}

	if bottom == 0 || bottom-top < 200 {
		return 0, 0, 0, 0
	}

	// Find left and right edges
	midY := (top + bottom) / 2
	left = 0
	right = width
	for x := 0; x < width/2; x++ {
		r, g, b, _ := img.At(x, midY).RGBA()
		r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
		if r8 < 235 || g8 < 235 || b8 < 235 {
			left = x
			break
		}
	}
	for x := width - 1; x > width/2; x-- {
		r, g, b, _ := img.At(x, midY).RGBA()
		r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
		if r8 < 235 || g8 < 235 || b8 < 235 {
			right = x + 1
			break
		}
	}

	return top, bottom, left, right
}

// findContentBounds finds the main content area (between status bar and nav bar)
// For text-only questions without illustrations
func findContentBounds(img image.Image) (top, bottom, left, right int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Find the top separator line (thin full-width line around y=400-440)
	// This marks the beginning of the question content area
	top = 0
	for y := 350; y < 500; y++ {
		nonWhite := 0
		total := 0
		for x := 50; x < width-50; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			total++
			if r8 < 230 || g8 < 230 || b8 < 230 {
				nonWhite++
			}
		}
		ratio := float64(nonWhite) / float64(total)
		if ratio > 0.95 {
			top = y
			break
		}
	}

	if top == 0 {
		// Fallback: use fixed crop for status bar
		top = 380
	}

	// Find bottom content boundary - scan from bottom upward
	// Look for the navigation bar area (buttons at bottom)
	bottom = height
	for y := height - 1; y > height/2; y-- {
		nonWhite := 0
		total := 0
		for x := 80; x < width-80; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			total++
			if r8 < 230 || g8 < 230 || b8 < 230 {
				nonWhite++
			}
		}
		ratio := float64(nonWhite) / float64(total)
		// Found content (not empty/nav area)
		if ratio > 0.05 {
			bottom = y + 1
			break
		}
	}

	// Find the answer buttons area at the bottom
	// Scan from bottom content up to find where answer options end
	// The content area typically ends where the last answer option text is
	// We keep everything between the separator and the bottom content

	left = 30
	right = width - 30

	return top, bottom, left, right
}

func main() {
	sourceDir := ".."

	// Load the logo
	logoFile, err := os.Open("logo.png")
	if err != nil {
		fmt.Printf("Error opening logo: %v\n", err)
		return
	}
	logoImg, err := png.Decode(logoFile)
	logoFile.Close()
	if err != nil {
		fmt.Printf("Error decoding logo: %v\n", err)
		return
	}

	processed := 0
	errors := 0

	// Find all Bilet directories
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		fmt.Printf("Error reading source dir: %v\n", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "Bilet") {
			continue
		}

		dirPath := filepath.Join(sourceDir, entry.Name())
		files, err := os.ReadDir(dirPath)
		if err != nil {
			fmt.Printf("Error reading dir %s: %v\n", entry.Name(), err)
			continue
		}

		fmt.Printf("\n--- %s ---\n", entry.Name())

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := strings.ToLower(file.Name())
			if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") && !strings.HasSuffix(name, ".png") {
				continue
			}

			imgPath := filepath.Join(dirPath, file.Name())

			// Open image
			f, err := os.Open(imgPath)
			if err != nil {
				fmt.Printf("  Error opening %s: %v\n", file.Name(), err)
				errors++
				continue
			}

			img, _, err := image.Decode(f)
			f.Close()
			if err != nil {
				fmt.Printf("  Error decoding %s: %v\n", file.Name(), err)
				errors++
				continue
			}

			// Try to find illustration bounds first
			cropTop, cropBottom, cropLeft, cropRight := findIllustrationBounds(img)

			cropType := "illustration"
			if cropTop == 0 {
				// No illustration found - use content bounds (text-only question)
				cropTop, cropBottom, cropLeft, cropRight = findContentBounds(img)
				cropType = "content"
			}

			// Crop
			croppedWidth := cropRight - cropLeft
			croppedHeight := cropBottom - cropTop

			if croppedWidth < 100 || croppedHeight < 100 {
				fmt.Printf("  Error: crop too small for %s (%dx%d)\n", file.Name(), croppedWidth, croppedHeight)
				errors++
				continue
			}

			croppedRGBA := image.NewRGBA(image.Rect(0, 0, croppedWidth, croppedHeight))
			draw.Draw(croppedRGBA, croppedRGBA.Bounds(), img, image.Point{cropLeft, cropTop}, draw.Src)

			// Place logo in the bottom-right corner
			logoSize := int(float64(croppedWidth) * 0.15)
			if logoSize < 60 {
				logoSize = 60
			}
			if logoSize > croppedHeight/3 {
				logoSize = croppedHeight / 3
			}

			// Resize logo
			logoBounds := logoImg.Bounds()
			scaledLogo := image.NewRGBA(image.Rect(0, 0, logoSize, logoSize))
			for y := 0; y < logoSize; y++ {
				for x := 0; x < logoSize; x++ {
					srcX := x * logoBounds.Dx() / logoSize
					srcY := y * logoBounds.Dy() / logoSize
					scaledLogo.Set(x, y, logoImg.At(srcX+logoBounds.Min.X, srcY+logoBounds.Min.Y))
				}
			}

			// Place logo at bottom-right
			margin := 10
			logoX := croppedWidth - logoSize - margin
			logoY := croppedHeight - logoSize - margin
			logoRect := image.Rect(logoX, logoY, logoX+logoSize, logoY+logoSize)
			draw.Draw(croppedRGBA, logoRect, scaledLogo, image.Point{0, 0}, draw.Over)

			// Save
			out, err := os.Create(imgPath)
			if err != nil {
				fmt.Printf("  Error creating %s: %v\n", file.Name(), err)
				errors++
				continue
			}
			err = jpeg.Encode(out, croppedRGBA, &jpeg.Options{Quality: 95})
			out.Close()
			if err != nil {
				fmt.Printf("  Error encoding %s: %v\n", file.Name(), err)
				errors++
				continue
			}

			processed++
			fmt.Printf("  OK [%s]: %s (%dx%d)\n", cropType, file.Name(), croppedWidth, croppedHeight)
		}
	}

	fmt.Printf("\n=== Done! Processed: %d, Errors: %d ===\n", processed, errors)
}
