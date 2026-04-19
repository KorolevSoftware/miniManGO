package main

import "github.com/go-gl/mathgl/mgl32"

type BoundBox struct {
	Min mgl32.Vec3
	Max mgl32.Vec3
}

func (bound *BoundBox) Contains(point Sample) bool {
	return point.X >= bound.Min.X() && point.X <= bound.Max.X() &&
		point.Y >= bound.Min.Y() && point.Y <= bound.Max.Y()
}

func (b BoundBox) Intersects(other BoundBox) bool {
	return (b.Min.X() <= other.Max.X() && b.Max.X() >= other.Min.X()) &&
		(b.Min.Y() <= other.Max.Y() && b.Max.Y() >= other.Min.Y()) &&
		(b.Min.Z() <= other.Max.Z() && b.Max.Z() >= other.Min.Z())
}
