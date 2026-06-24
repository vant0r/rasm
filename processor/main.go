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

const (
	outputWidth  = 1016
	outputHeight = 550
)

func findIllustrationBounds(img image.Image) (top, bottom, left, right int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

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

	if bottom == 0 || bottom-top < 150 {
		return 0, 0, 0, 0
	}

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

func resizeToFit(src image.Image, srcLeft, srcTop, srcRight, srcBottom int) *image.RGBA {
	srcW := srcRight - srcLeft
	srcH := srcBottom - srcTop

	scaleX := float64(outputWidth) / float64(srcW)
	scaleY := float64(outputHeight) / float64(srcH)

	scale := scaleX
	if scaleY > scaleX {
		scale = scaleY
	}

	scaledW := int(float64(srcW) * scale)
	scaledH := int(float64(srcH) * scale)

	offsetX := (scaledW - outputWidth) / 2
	offsetY := (scaledH - outputHeight) / 2

	dst := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))

	for y := 0; y < outputHeight; y++ {
		for x := 0; x < outputWidth; x++ {
			sx := srcLeft + int(float64(x+offsetX)/scale)
			sy := srcTop + int(float64(y+offsetY)/scale)
			if sx >= srcLeft && sx < srcRight && sy >= srcTop && sy < srcBottom {
				dst.Set(x, y, src.At(sx, sy))
			}
		}
	}

	return dst
}

func main() {
	sourceDir := ".."

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

	logoSizeF := float64(outputWidth) * 0.14
	logoSize := int(logoSizeF)
	logoBounds := logoImg.Bounds()
	scaledLogo := image.NewRGBA(image.Rect(0, 0, logoSize, logoSize))
	for y := 0; y < logoSize; y++ {
		for x := 0; x < logoSize; x++ {
			srcX := x * logoBounds.Dx() / logoSize
			srcY := y * logoBounds.Dy() / logoSize
			scaledLogo.Set(x, y, logoImg.At(srcX+logoBounds.Min.X, srcY+logoBounds.Min.Y))
		}
	}

	processed := 0
	skipped := 0
	errors := 0

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "Bilet") {
			continue
		}

		dirPath := filepath.Join(sourceDir, entry.Name())
		files, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := strings.ToLower(file.Name())
			if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") {
				continue
			}

			imgPath := filepath.Join(dirPath, file.Name())

			f, err := os.Open(imgPath)
			if err != nil {
				errors++
				continue
			}

			img, _, err := image.Decode(f)
			f.Close()
			if err != nil {
				errors++
				continue
			}

			cropTop, cropBottom, cropLeft, cropRight := findIllustrationBounds(img)
			if cropTop == 0 {
				skipped++
				continue
			}

			result := resizeToFit(img, cropLeft, cropTop, cropRight, cropBottom)

			margin := 8
			logoX := outputWidth - logoSize - margin
			logoY := outputHeight - logoSize - margin
			logoRect := image.Rect(logoX, logoY, logoX+logoSize, logoY+logoSize)
			draw.Draw(result, logoRect, scaledLogo, image.Point{0, 0}, draw.Over)

			out, err := os.Create(imgPath)
			if err != nil {
				errors++
				continue
			}
			jpeg.Encode(out, result, &jpeg.Options{Quality: 95})
			out.Close()

			processed++
		}
		fmt.Printf("%s: done\n", entry.Name())
	}

	fmt.Printf("\n=== Processed: %d, Skipped: %d, Errors: %d ===\n", processed, skipped, errors)
}
