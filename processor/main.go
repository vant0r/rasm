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
	// It spans nearly the full width (from ~x=30 to ~x=1050 on 1080px wide images)
	// We scan from y=500 downward looking for rows where >70% of central pixels are non-white
	// The illustration content fills most of the width unlike text (which only fills ~10-30%)

	// Step 1: Find the top of illustration
	// Look for the first row where most of the width is filled with content
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
				// The illustration started 30 rows ago
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

	// Step 2: Find the bottom of illustration
	// Scan from top downward until we hit sustained white area
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

	// Step 3: Find left and right edges
	// Scan at the vertical midpoint of the illustration
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

func main() {
	dirs := []string{"Bilet 1", "Bilet 2", "Bilet 3", "Bilet 4", "Bilet 5"}

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
	skipped := 0
	errors := 0

	for _, dir := range dirs {
		dirPath := filepath.Join("..", dir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			fmt.Printf("Error reading dir %s: %v\n", dir, err)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := strings.ToLower(entry.Name())
			if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") && !strings.HasSuffix(name, ".png") {
				continue
			}

			imgPath := filepath.Join(dirPath, entry.Name())

			// Open image
			f, err := os.Open(imgPath)
			if err != nil {
				fmt.Printf("Error opening %s: %v\n", imgPath, err)
				errors++
				continue
			}

			img, _, err := image.Decode(f)
			f.Close()
			if err != nil {
				fmt.Printf("Error decoding %s: %v\n", imgPath, err)
				errors++
				continue
			}

			// Find illustration bounds
			cropTop, cropBottom, cropLeft, cropRight := findIllustrationBounds(img)
			if cropTop == 0 {
				fmt.Printf("Skipped (no illustration): %s\n", imgPath)
				skipped++
				continue
			}

			// Crop the illustration
			croppedRGBA := image.NewRGBA(image.Rect(0, 0, cropRight-cropLeft, cropBottom-cropTop))
			draw.Draw(croppedRGBA, croppedRGBA.Bounds(), img, image.Point{cropLeft, cropTop}, draw.Src)

			// Place logo in the bottom-right corner
			// Scale logo to appropriate size (about 15% of the cropped image width)
			croppedWidth := cropRight - cropLeft
			croppedHeight := cropBottom - cropTop
			logoSize := int(float64(croppedWidth) * 0.15)
			if logoSize < 60 {
				logoSize = 60
			}

			// Resize logo using simple nearest-neighbor (good enough for a circular logo)
			logoBounds := logoImg.Bounds()
			scaledLogo := image.NewRGBA(image.Rect(0, 0, logoSize, logoSize))
			for y := 0; y < logoSize; y++ {
				for x := 0; x < logoSize; x++ {
					srcX := x * logoBounds.Dx() / logoSize
					srcY := y * logoBounds.Dy() / logoSize
					scaledLogo.Set(x, y, logoImg.At(srcX+logoBounds.Min.X, srcY+logoBounds.Min.Y))
				}
			}

			// Place logo at bottom-right with small margin
			margin := 10
			logoX := croppedWidth - logoSize - margin
			logoY := croppedHeight - logoSize - margin
			logoRect := image.Rect(logoX, logoY, logoX+logoSize, logoY+logoSize)
			draw.Draw(croppedRGBA, logoRect, scaledLogo, image.Point{0, 0}, draw.Over)

			// Save
			out, err := os.Create(imgPath)
			if err != nil {
				fmt.Printf("Error creating %s: %v\n", imgPath, err)
				errors++
				continue
			}
			err = jpeg.Encode(out, croppedRGBA, &jpeg.Options{Quality: 95})
			out.Close()
			if err != nil {
				fmt.Printf("Error encoding %s: %v\n", imgPath, err)
				errors++
				continue
			}

			processed++
			fmt.Printf("Processed: %s (crop: %d,%d -> %d,%d = %dx%d)\n",
				imgPath, cropLeft, cropTop, cropRight, cropBottom, croppedWidth, croppedHeight)
		}
	}

	fmt.Printf("\nDone! Processed: %d, Skipped: %d, Errors: %d\n", processed, skipped, errors)
}
