package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/lucasb-eyer/go-colorful"
)

const lutSize = 33
const skinToneHueStart = 14.0 // Skin tone hue range start
const skinToneHueEnd = 32.0   // Skin tone hue range end
const hueShift = 5.0          // Range for cold and warm neighboring hues

// Determines if a hue should be replaced by pure yellow, green, or magenta
func replaceColor(h float64) colorful.Color {
	mid := skinToneHueStart + (skinToneHueEnd-skinToneHueStart)/2

	if h < skinToneHueStart || h > skinToneHueEnd {
		return colorful.Color{}
	}

	if h >= mid-hueShift && h <= mid+hueShift {
		return colorful.Color{R: 0.0, G: 0.0, B: 1.0} // blue
	}
	if h > mid {
		return colorful.Color{R: 0.0, G: 1.0, B: 0.0} // green
	}
	return colorful.Color{R: 1.0, G: 0.0, B: 1.0} // magenta
}

// Writes a .cube LUT file
func writeLUT(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write LUT header
	writer.WriteString("TITLE \"Skin Tone to Yellow, Green, Magenta LUT\"\n")
	writer.WriteString(fmt.Sprintf("LUT_3D_SIZE %d\n", lutSize))
	writer.WriteString("DOMAIN_MIN 0.0 0.0 0.0\n")
	writer.WriteString("DOMAIN_MAX 1.0 1.0 1.0\n")

	// Generate LUT values (RGB triplets)
	for r := 0; r < lutSize; r++ {
		for g := 0; g < lutSize; g++ {
			for b := 0; b < lutSize; b++ {
				// Normalize RGB to [0, 1] range
				rNorm := float64(r) / float64(lutSize-1)
				gNorm := float64(g) / float64(lutSize-1)
				bNorm := float64(b) / float64(lutSize-1)

				// Create a colorful.Color from normalized RGB values
				color := colorful.Color{R: rNorm, G: gNorm, B: bNorm}

				// Convert RGB to HSL to check hue
				h, _, _ := color.Hsl()

				// Get the replacement color (if applicable)
				replacementColor := replaceColor(h) // Convert to [0, 360] range

				// If the color is replaced, use the new color, otherwise, keep the original color
				if replacementColor != (colorful.Color{}) {
					color = replacementColor
				}

				// Write the final RGB values to the LUT file
				writer.WriteString(fmt.Sprintf("%.6f %.6f %.6f\n", color.R, color.G, color.B))
			}
		}
	}

	// Flush the writer to ensure all data is written
	return writer.Flush()
}

func main() {
	lutFilename := "false-skin.cube"

	// Generate LUT based on skin tone replacement logic
	err := writeLUT(lutFilename)
	if err != nil {
		fmt.Printf("Error writing LUT: %v\n", err)
		return
	}

	fmt.Printf("LUT written to %s\n", lutFilename)
}
