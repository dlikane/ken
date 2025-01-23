package imgalign

import (
	"gocv.io/x/gocv"
	"image"
)

// TranslateImage applies translation to the input image based on dx and dy.
func TranslateImage(img gocv.Mat, dx, dy int) gocv.Mat {
	// Create a translation matrix
	translationMatrix := gocv.NewMatWithSize(2, 3, gocv.MatTypeCV64F)
	defer translationMatrix.Close()

	// Fill the translation matrix
	translationMatrix.SetDoubleAt(0, 0, 1) // [1, 0, dx]
	translationMatrix.SetDoubleAt(0, 1, 0)
	translationMatrix.SetDoubleAt(0, 2, float64(dx))
	translationMatrix.SetDoubleAt(1, 0, 0) // [0, 1, dy]
	translationMatrix.SetDoubleAt(1, 1, 1)
	translationMatrix.SetDoubleAt(1, 2, float64(dy))

	// Apply the translation using WarpAffine
	translated := gocv.NewMat()

	size := img.Size()
	destSize := image.Point{X: size[1], Y: size[0]} // Width, Height

	gocv.WarpAffine(img, &translated, translationMatrix, destSize)

	return translated
}
