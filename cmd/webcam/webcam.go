package main

import (
	"fmt"
	"gocv.io/x/gocv"
	"log"
)

func main() {
	fmt.Println("GoCV Version:", gocv.Version())
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		log.Fatalf("Error opening webcam: %v", err)
	}
	defer webcam.Close()
	window := gocv.NewWindow("GoCV Test")
	defer window.Close()
	img := gocv.NewMat()
	defer img.Close()
	for {
		if ok := webcam.Read(&img); !ok {
			log.Println("Cannot read from webcam")
			break
		}
		window.IMShow(img)
		if window.WaitKey(1) >= 0 {
			break
		}
	}
}
