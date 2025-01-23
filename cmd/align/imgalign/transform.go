package imgalign

import (
	"fmt"
	"image"
	"math"
)

// CalculateTransform calculates rotation, scaling, and translation needed to align two images.
func CalculateTransform(baseEyes, targetEyes [2]image.Point) (angle, scale, dx, dy float64) {
	// Calculate the distance between eyes for base and target images
	baseDist := math.Hypot(float64(baseEyes[1].X-baseEyes[0].X), float64(baseEyes[1].Y-baseEyes[0].Y))
	targetDist := math.Hypot(float64(targetEyes[1].X-targetEyes[0].X), float64(targetEyes[1].Y-targetEyes[0].Y))

	// Calculate the angle between the eyes for base and target images
	baseAngle := math.Atan2(float64(baseEyes[1].Y-baseEyes[0].Y), float64(baseEyes[1].X-baseEyes[0].X))
	targetAngle := math.Atan2(float64(targetEyes[1].Y-targetEyes[0].Y), float64(targetEyes[1].X-targetEyes[0].X))
	angle = -(baseAngle - targetAngle)

	// Calculate the scaling factor
	scale = baseDist / targetDist

	dx = float64(baseEyes[0].X - targetEyes[0].X)
	dy = float64(baseEyes[0].Y - targetEyes[0].Y)

	// Debug output
	fmt.Printf("Turn : base: %v target: %v turn: %v\n", baseAngle, targetAngle, angle)
	fmt.Printf("Scale: base: %v target: %v scale: %v\n", baseDist, targetDist, scale)
	fmt.Printf("MoveX: base: %v target: %v dx: %v\n", baseEyes[0].X, targetEyes[0].X, dx)
	fmt.Printf("MoveY: base: %v target: %v dx: %v\n", baseEyes[0].Y, targetEyes[0].Y, dy)
	return angle, scale, dx, dy
}
