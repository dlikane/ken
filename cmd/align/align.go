package main

import (
	"gocv.io/x/gocv"
	"log"
	"os"
	"path/filepath"
	"strings"

	"ken/cmd/align/imgalign"
)

// Command line: align data/base.jpg models frames aligned
func main() {
	isLocal := true
	baseImagePath := "data/base.jpg"
	modelsDir := "models"
	inputDir := "frames"
	outputDir := "aligned"

	if len(os.Args) >= 5 {
		isLocal = false
		baseImagePath = os.Args[1]
		modelsDir = os.Args[2]
		inputDir = os.Args[3]
		outputDir = os.Args[4]
	}

	// Detect landmarks in the base image
	baseEyes, err := imgalign.DetectLandmarks(isLocal, baseImagePath, filepath.Join(modelsDir, "shape_predictor_68_face_landmarks.dat"))
	if err != nil {
		log.Fatalf("Failed to detect landmarks in base image: %v", err)
	}

	// Process all images in the input directory
	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".jpg") {
			return nil
		}

		log.Printf("Processing %s...", path)

		// Detect landmarks in the target image
		targetEyes, err := imgalign.DetectLandmarks(isLocal, path, filepath.Join(modelsDir, "shape_predictor_68_face_landmarks.dat"))
		if err != nil {
			log.Printf("Failed to detect landmarks in %s: %v", path, err)
			return nil
		}

		// Load the target image
		targetImage := gocv.IMRead(path, gocv.IMReadColor)
		if targetImage.Empty() {
			log.Printf("Failed to read image: %s", path)
			return nil
		}
		defer targetImage.Close()

		// Calculate transformation and align the image
		angle, scale, dx, dy := imgalign.CalculateTransform(baseEyes, targetEyes)
		alignedImage := imgalign.AlignImage(targetEyes, targetImage, angle, scale, dx, dy)
		defer alignedImage.Close()

		// Save the aligned image to the output directory
		outputPath := filepath.Join(outputDir, info.Name())
		if ok := gocv.IMWrite(outputPath, alignedImage); !ok {
			log.Printf("Failed to write aligned image to %s", outputPath)
		} else {
			log.Printf("Aligned image saved to %s", outputPath)
		}

		return nil
	})

	if err != nil {
		log.Fatalf("Failed to process images: %v", err)
	}

	log.Println("Processing completed.")
}
