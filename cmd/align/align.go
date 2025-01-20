package main

import (
	"fmt"
	"gocv.io/x/gocv"
	"image"
	"log"
	"math"
)

// DetectFaces detects faces in an image using a Haar Cascade.
func DetectFaces(imagePath string, cascadeFile string) ([]image.Rectangle, error) {
	img := gocv.IMRead(imagePath, gocv.IMReadColor)
	defer img.Close()

	cascade := gocv.NewCascadeClassifier()
	defer cascade.Close()

	if !cascade.Load(cascadeFile) {
		return nil, fmt.Errorf("failed to load Haar cascade file: %s", cascadeFile)
	}

	rects := cascade.DetectMultiScale(img)
	return rects, nil
}

// DetectLandmarks detects facial landmarks using Dlib.
func DetectLandmarks(imagePath string, modelPath string) ([2]image.Point, error) {
	// TODO: Use Go bindings for Dlib to extract landmarks
	// Implement logic to call Dlib's `shape_predictor_68_face_landmarks.dat` model
	// Replace the dummy return below with real landmark positions.

	// Dummy values for demonstration purposes
	leftEye := image.Point{X: 100, Y: 150}
	rightEye := image.Point{X: 200, Y: 150}

	return [2]image.Point{leftEye, rightEye}, nil
}

// CalculateTransform calculates rotation, scaling, and translation needed to align two images.
func CalculateTransform(baseEyes, targetEyes [2]image.Point) (angle, scale float64, dx, dy float64) {
	baseDist := math.Hypot(float64(baseEyes[1].X-baseEyes[0].X), float64(baseEyes[1].Y-baseEyes[0].Y))
	targetDist := math.Hypot(float64(targetEyes[1].X-targetEyes[0].X), float64(targetEyes[1].Y-targetEyes[0].Y))
	angle = math.Atan2(float64(baseEyes[1].Y-baseEyes[0].Y), float64(baseEyes[1].X-baseEyes[0].X)) -
		math.Atan2(float64(targetEyes[1].Y-targetEyes[0].Y), float64(targetEyes[1].X-targetEyes[0].X))
	scale = baseDist / targetDist
	dx = float64(baseEyes[0].X) - float64(targetEyes[0].X)*scale
	dy = float64(baseEyes[0].Y) - float64(targetEyes[0].Y)*scale
	return angle, scale, dx, dy
}

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

// AlignImage applies rotation, scaling, and translation to align the target image.
func AlignImage(targetImage gocv.Mat, angle, scale, dx, dy float64) gocv.Mat {
	// Apply scaling
	scaled := gocv.NewMat()
	gocv.Resize(targetImage, &scaled, image.Point{}, scale, scale, gocv.InterpolationLinear)

	// Apply rotation
	center := image.Point{X: scaled.Cols() / 2, Y: scaled.Rows() / 2}
	rotated := gocv.NewMat()
	rotationMatrix := gocv.GetRotationMatrix2D(center, angle*180/math.Pi, 1.0)
	size := scaled.Size()
	destSize := image.Point{X: size[1], Y: size[0]} // Width, Height

	gocv.WarpAffine(scaled, &rotated, rotationMatrix, destSize)

	// Apply translation
	translated := TranslateImage(rotated, int(dx), int(dy))

	return translated
}

func main() {
	// Paths to input images and models
	basePath := "data/base.jpg"
	targetPath := "data/target.jpg"
	cascadePath := "models/haarcascade_frontalface_default.xml"
	landmarkModelPath := "models/shape_predictor_68_face_landmarks.dat"
	outputPath := "data/aligned.jpg"

	// Detect faces and landmarks in the base image
	baseFaces, err := DetectFaces(basePath, cascadePath)
	if err != nil {
		log.Fatalf("Failed to detect faces in base image: %v", err)
	}
	if len(baseFaces) == 0 {
		log.Fatalf("No faces detected in base image")
	}

	baseEyes, err := DetectLandmarks(basePath, landmarkModelPath)
	if err != nil {
		log.Fatalf("Failed to detect landmarks in base image: %v", err)
	}

	// Detect faces and landmarks in the target image
	targetFaces, err := DetectFaces(targetPath, cascadePath)
	if err != nil {
		log.Fatalf("Failed to detect faces in target image: %v", err)
	}
	if len(targetFaces) == 0 {
		log.Fatalf("No faces detected in target image")
	}

	targetEyes, err := DetectLandmarks(targetPath, landmarkModelPath)
	if err != nil {
		log.Fatalf("Failed to detect landmarks in target image: %v", err)
	}

	// Calculate transformation
	angle, scale, dx, dy := CalculateTransform(baseEyes, targetEyes)

	// Align target image
	targetImg := gocv.IMRead(targetPath, gocv.IMReadColor)
	defer targetImg.Close()

	alignedImage := AlignImage(targetImg, angle, scale, dx, dy)

	// Save aligned image
	if ok := gocv.IMWrite(outputPath, alignedImage); !ok {
		log.Fatalf("Failed to save aligned image")
	}

	log.Printf("Aligned image saved to %s", outputPath)
}
