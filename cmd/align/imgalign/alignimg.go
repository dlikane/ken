package imgalign

import (
	"gocv.io/x/gocv"
	"image"
	"log"
	"math"
)

// AlignImage applies rotation, scaling, and translation to align the target image.
func AlignImage(targetEyes [2]image.Point, img gocv.Mat, angle, scale, dx, dy float64) gocv.Mat {
	// Get the original size of the canvas
	canvasSize := img.Size()
	canvasWidth := canvasSize[1]
	canvasHeight := canvasSize[0]

	// Get the original position of the left eye
	originalLeftEye := targetEyes[0]

	// Scale the image
	scaled := gocv.NewMat()
	newWidth := int(float64(img.Cols()) * scale)
	newHeight := int(float64(img.Rows()) * scale)
	gocv.Resize(img, &scaled, image.Point{X: newWidth, Y: newHeight}, 0, 0, gocv.InterpolationLinear)

	// Calculate the new position of the left eye after scaling
	scaledLeftEyeX := int(float64(originalLeftEye.X) * scale)
	scaledLeftEyeY := int(float64(originalLeftEye.Y) * scale)

	// Create a blank canvas with the original image size
	canvas := gocv.NewMatWithSize(canvasHeight, canvasWidth, img.Type())
	defer canvas.Close()

	// Calculate offsets to keep the scaled left eye at its original position
	offsetX := originalLeftEye.X - scaledLeftEyeX
	offsetY := originalLeftEye.Y - scaledLeftEyeY

	// Clamp offsets and adjust ROI to fit within the canvas
	roiX := max(0, offsetX)
	roiY := max(0, offsetY)
	roiWidth := min(newWidth, canvasWidth-roiX)
	roiHeight := min(newHeight, canvasHeight-roiY)

	// Ensure ROI dimensions are valid
	if roiWidth <= 0 || roiHeight <= 0 {
		log.Println("Scaled image does not fit into the canvas.")
		return canvas
	}

	// Place the scaled image onto the canvas
	scaledROI := canvas.Region(image.Rect(roiX, roiY, roiX+roiWidth, roiY+roiHeight))
	defer scaledROI.Close()
	scaledRegion := scaled.Region(image.Rect(max(0, -offsetX), max(0, -offsetY), max(0, -offsetX)+roiWidth, max(0, -offsetY)+roiHeight))
	defer scaledRegion.Close()
	scaledRegion.CopyTo(&scaledROI)

	// Apply rotation
	center := originalLeftEye // Rotate around the left eye
	rotated := gocv.NewMat()
	rotationMatrix := gocv.GetRotationMatrix2D(center, angle*180/math.Pi, 1.0)
	gocv.WarpAffine(canvas, &rotated, rotationMatrix, image.Point{X: canvasWidth, Y: canvasHeight})

	// Apply translation relative to the original canvas
	translated := TranslateImage(rotated, int(dx), int(dy))

	return translated
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
