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
	"strings"
)

const (
	// Crop parameters - for phone screenshots
	// These are tuned for 1080x2400 screenshots (common Android resolution)
	topCrop    = 120 // Status bar (clock, battery, etc.)
	bottomCrop = 160 // Navigation bar (back, home, recent buttons)

	// Logo size relative to image width
	logoSizeRatio  = 0.18 // Logo will be 18% of image width
	logoMarginRatio = 0.02 // Margin from edges

	// Source directory
	sourceDir = "/projects/sandbox/rasm"
	logoPath  = "/projects/sandbox/rasm/processor/logo.png"
)

// Vatanparvar logo colors
var (
	darkBlue  = color.NRGBA{R: 20, G: 50, B: 120, A: 255}   // Dark navy blue outer ring
	goldColor = color.NRGBA{R: 200, G: 170, B: 80, A: 255}   // Gold/tan color
	lightGold = color.NRGBA{R: 220, G: 190, B: 100, A: 255}  // Lighter gold
	skyBlue   = color.NRGBA{R: 100, G: 180, B: 230, A: 255}  // Light blue background
	darkGold  = color.NRGBA{R: 180, G: 150, B: 60, A: 255}   // Darker gold for outlines
	whiteCol  = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	blackCol  = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
)

// createVatanparvarLogo creates a circular logo matching the Vatanparvar emblem
// Blue outer ring with gold text area, light blue center with map and wheat
func createVatanparvarLogo(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))

	center := float64(size) / 2
	radius := float64(size) / 2.0

	// Fill with transparency
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, color.NRGBA{0, 0, 0, 0})
		}
	}

	// Draw layers from outside to inside
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist > radius {
				continue
			}

			// Gold outer border (thin)
			if dist >= radius*0.95 {
				img.Set(x, y, goldColor)
				continue
			}

			// Dark blue ring (main text ring)
			if dist >= radius*0.75 {
				img.Set(x, y, darkBlue)

				// Add gold dots/decorations in the blue ring
				angle := math.Atan2(dy, dx)
				// Small gold dots at bottom
				if angle > 1.3 && angle < 1.85 {
					segAngle := math.Mod(angle*20, math.Pi*2)
					if segAngle < 0.3 {
						img.Set(x, y, goldColor)
					}
				}
				continue
			}

			// Gold inner border
			if dist >= radius*0.72 {
				img.Set(x, y, goldColor)
				continue
			}

			// Light blue inner circle (sky)
			if dist < radius*0.72 {
				// Gradient from light blue at top to slightly darker at bottom
				t := (dy + center) / (2 * center)
				r := uint8(100 + int(float64(150-100)*t))
				g := uint8(180 + int(float64(210-180)*t))
				b := uint8(230 + int(float64(250-230)*t))
				img.Set(x, y, color.NRGBA{r, g, b, 255})

				// Draw sun rays at the top
				if dy < -radius*0.2 {
					sunAngle := math.Atan2(dy+radius*0.3, dx)
					sunDist := math.Sqrt(dx*dx + (dy+radius*0.3)*(dy+radius*0.3))
					if sunDist < radius*0.5 {
						rayAngle := math.Mod(sunAngle+math.Pi, math.Pi/8)
						if rayAngle < math.Pi/16 {
							img.Set(x, y, color.NRGBA{255, 230, 150, 180})
						}
					}
				}

				// Draw Uzbekistan map outline (simplified as a shape in center)
				mapCenterY := center - radius*0.05
				mapDx := dx / (radius * 0.4)
				mapDy := (float64(y) - mapCenterY) / (radius * 0.25)

				// Simplified map shape
				if math.Abs(mapDx) < 1.0 && math.Abs(mapDy) < 1.0 {
					// Create an irregular shape resembling Uzbekistan
					inMap := false
					if mapDy > -0.8 && mapDy < 0.7 {
						leftBound := -0.8 + mapDy*0.2
						rightBound := 0.9 - mapDy*0.1
						if mapDx > leftBound && mapDx < rightBound {
							// Add some irregularity
							noise := math.Sin(mapDx*5)*0.1 + math.Cos(mapDy*4)*0.1
							if mapDx > leftBound+noise && mapDx < rightBound+noise {
								inMap = true
							}
						}
					}
					if inMap {
						img.Set(x, y, color.NRGBA{220, 200, 140, 200})
					}
				}

				// Draw wheat stalks at bottom
				if dy > radius*0.15 {
					// Left wheat stalk
					leftStalkAngle := math.Pi/2 + math.Pi/5
					stalkDx := dx - float64(size)*0.0
					stalkDy := dy - radius*0.3
					rotX := stalkDx*math.Cos(-leftStalkAngle+math.Pi/2) - stalkDy*math.Sin(-leftStalkAngle+math.Pi/2)
					rotY := stalkDx*math.Sin(-leftStalkAngle+math.Pi/2) + stalkDy*math.Cos(-leftStalkAngle+math.Pi/2)

					if math.Abs(rotX) < radius*0.25 && math.Abs(rotY) < radius*0.03 {
						img.Set(x, y, goldColor)
					}

					// Right wheat stalk (mirror)
					rightStalkAngle := math.Pi/2 - math.Pi/5
					rotX2 := stalkDx*math.Cos(-rightStalkAngle+math.Pi/2) - stalkDy*math.Sin(-rightStalkAngle+math.Pi/2)
					rotY2 := stalkDx*math.Sin(-rightStalkAngle+math.Pi/2) + stalkDy*math.Cos(-rightStalkAngle+math.Pi/2)

					if math.Abs(rotX2) < radius*0.25 && math.Abs(rotY2) < radius*0.03 {
						img.Set(x, y, goldColor)
					}

					// Wheat grains (oval shapes along stalks)
					for i := 0; i < 6; i++ {
						grainOffset := float64(i) * radius * 0.06
						grainX := -radius*0.15 - grainOffset*0.7
						grainY := radius*0.2 + grainOffset*0.5
						grainDist := math.Sqrt(math.Pow(dx-grainX, 2)/4 + math.Pow(dy-grainY, 2))
						if grainDist < radius*0.04 {
							img.Set(x, y, goldColor)
						}
						// Mirror for right side
						grainDist2 := math.Sqrt(math.Pow(dx+grainX, 2)/4 + math.Pow(dy-grainY, 2))
						if grainDist2 < radius*0.04 {
							img.Set(x, y, goldColor)
						}
					}
				}
			}
		}
	}

	// Draw text simulation in the blue ring (white dots/marks)
	for angle := -2.8; angle < -0.3; angle += 0.08 {
		textR := radius * 0.85
		tx := center + textR*math.Cos(angle)
		ty := center + textR*math.Sin(angle)
		// Draw small white rectangle for each "letter"
		for dy := -2.0; dy <= 2.0; dy++ {
			for dx := -1.5; dx <= 1.5; dx++ {
				px := int(tx + dx)
				py := int(ty + dy)
				if px >= 0 && px < size && py >= 0 && py < size {
					img.Set(px, py, color.NRGBA{220, 200, 100, 230})
				}
			}
		}
	}

	// Bottom text arc
	for angle := 0.3; angle < 2.8; angle += 0.08 {
		textR := radius * 0.85
		tx := center + textR*math.Cos(angle)
		ty := center + textR*math.Sin(angle)
		for dy := -2.0; dy <= 2.0; dy++ {
			for dx := -1.5; dx <= 1.5; dx++ {
				px := int(tx + dx)
				py := int(ty + dy)
				if px >= 0 && px < size && py >= 0 && py < size {
					img.Set(px, py, color.NRGBA{220, 200, 100, 230})
				}
			}
		}
	}

	return img
}

func loadOrCreateLogo(targetSize int) (*image.RGBA, error) {
	// Try to load existing logo from file
	f, err := os.Open(logoPath)
	if err == nil {
		defer f.Close()
		img, err := png.Decode(f)
		if err == nil {
			bounds := img.Bounds()
			w := bounds.Dx()
			h := bounds.Dy()

			// If logo is already the right size, use it directly
			if w == targetSize && h == targetSize {
				rgba := image.NewRGBA(image.Rect(0, 0, w, h))
				draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)
				fmt.Printf("Loaded logo from file: %s (%dx%d)\n", logoPath, w, h)
				return rgba, nil
			}

			// Resize logo to target size using nearest-neighbor scaling
			rgba := resizeImage(img, targetSize, targetSize)
			fmt.Printf("Loaded and resized logo from %dx%d to %dx%d\n", w, h, targetSize, targetSize)
			return rgba, nil
		}
	}

	// Generate logo
	fmt.Printf("Generating Vatanparvar logo (%dx%d)...\n", targetSize, targetSize)
	logo := createVatanparvarLogo(targetSize)

	// Save for future use
	outFile, err := os.Create(logoPath)
	if err == nil {
		png.Encode(outFile, logo)
		outFile.Close()
		fmt.Println("Saved generated logo to:", logoPath)
	}

	return logo, nil
}

// resizeImage scales an image to the target width/height using bilinear interpolation
func resizeImage(src image.Image, targetW, targetH int) *image.RGBA {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))

	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			// Map destination coords to source coords
			srcX := float64(x) * float64(srcW) / float64(targetW)
			srcY := float64(y) * float64(srcH) / float64(targetH)

			// Bilinear interpolation
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

			r := bilinear(r00, r10, r01, r11, fx, fy)
			g := bilinear(g00, g10, g01, g11, fx, fy)
			b := bilinear(b00, b10, b01, b11, fx, fy)
			a := bilinear(a00, a10, a01, a11, fx, fy)

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

func bilinear(v00, v10, v01, v11 uint32, fx, fy float64) uint32 {
	top := float64(v00)*(1-fx) + float64(v10)*fx
	bottom := float64(v01)*(1-fx) + float64(v11)*fx
	return uint32(top*(1-fy) + bottom*fy)
}

func processImage(inputPath string, logo *image.RGBA) error {
	// Open the source image
	f, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	// Decode JPEG
	srcImg, err := jpeg.Decode(f)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	bounds := srcImg.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate crop rectangle
	cropTop := topCrop
	cropBottom := bottomCrop

	// Adjust crop values for different resolutions
	if origHeight > 2500 {
		cropTop = int(float64(origHeight) * 0.05)
		cropBottom = int(float64(origHeight) * 0.065)
	} else if origHeight < 2000 {
		cropTop = int(float64(origHeight) * 0.04)
		cropBottom = int(float64(origHeight) * 0.055)
	}

	cropRect := image.Rect(0, cropTop, origWidth, origHeight-cropBottom)
	newWidth := cropRect.Dx()
	newHeight := cropRect.Dy()

	// Create new image with cropped dimensions
	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Draw cropped source image
	draw.Draw(dst, dst.Bounds(), srcImg, image.Point{cropRect.Min.X, cropRect.Min.Y}, draw.Src)

	// Calculate logo size and position based on image dimensions
	logoSize := int(float64(newWidth) * logoSizeRatio)
	margin := int(float64(newWidth) * logoMarginRatio)

	// Resize logo if needed
	var logoToUse *image.RGBA
	logoBounds := logo.Bounds()
	if logoBounds.Dx() != logoSize {
		logoToUse = resizeImage(logo, logoSize, logoSize)
	} else {
		logoToUse = logo
	}

	// Position at bottom-right corner
	logoX := newWidth - logoSize - margin
	logoY := newHeight - logoSize - margin
	logoRect := image.Rect(logoX, logoY, logoX+logoSize, logoY+logoSize)

	// Draw logo with alpha blending (Over operation preserves transparency)
	draw.Draw(dst, logoRect, logoToUse, image.Point{0, 0}, draw.Over)

	// Close the input file before overwriting
	f.Close()

	// Save output (overwrite original)
	outFile, err := os.Create(inputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	// Encode as JPEG with high quality
	err = jpeg.Encode(outFile, dst, &jpeg.Options{Quality: 92})
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	return nil
}

func main() {
	fmt.Println("=== Vatanparvar Bilet Image Processor ===")
	fmt.Printf("Crop: top=%dpx, bottom=%dpx (base, auto-adjusts for resolution)\n", topCrop, bottomCrop)
	fmt.Printf("Logo: size=%.0f%% of width, margin=%.0f%% of width\n", logoSizeRatio*100, logoMarginRatio*100)
	fmt.Println()

	// Load or generate logo (at a base size, will be resized per-image)
	logo, err := loadOrCreateLogo(300)
	if err != nil {
		fmt.Printf("Error loading logo: %v\n", err)
		os.Exit(1)
	}

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

		fmt.Printf("\n--- %s (%d files) ---\n", entry.Name(), len(files))

		for _, file := range files {
			if file.IsDir() {
				continue
			}
			name := strings.ToLower(file.Name())
			if !strings.HasSuffix(name, ".jpg") && !strings.HasSuffix(name, ".jpeg") {
				continue
			}

			imgPath := filepath.Join(biletDir, file.Name())
			fmt.Printf("  %s ... ", file.Name())

			if err := processImage(imgPath, logo); err != nil {
				fmt.Printf("ERROR: %v\n", err)
				errCount++
			} else {
				fmt.Println("OK")
				processed++
			}
		}
	}

	fmt.Printf("\n=== Done! Processed: %d, Errors: %d ===\n", processed, errCount)
}

// Suppress unused import warnings
var _ = blackCol
var _ = whiteCol
var _ = lightGold
var _ = darkGold
