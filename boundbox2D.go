package main

import "github.com/go-gl/mathgl/mgl32"

type BoundBox2D struct {
	Min mgl32.Vec3
	Max mgl32.Vec3
}

func (bound *BoundBox2D) Contains(point Sample) bool {
	return point.X >= bound.Min.X() && point.X <= bound.Max.X() &&
		point.Y >= bound.Min.Y() && point.Y <= bound.Max.Y()
}
