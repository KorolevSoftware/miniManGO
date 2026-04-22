package main

import (
	"image/color"
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

type SplitAxis int

const (
	SplitAxisNone SplitAxis = iota
	SplitAxisU
	SplitAxisV
)

type BilinearPatch struct {
	P00   mgl32.Vec3
	P01   mgl32.Vec3
	P10   mgl32.Vec3
	P11   mgl32.Vec3
	UV00  mgl32.Vec2
	UV01  mgl32.Vec2
	UV10  mgl32.Vec2
	UV11  mgl32.Vec2
	Color color.Color
}

func (bp *BilinearPatch) Project(projectFunc func(mgl32.Vec3) mgl32.Vec3) BilinearPatch {
	projectPach := *bp
	projectPach.P00 = projectFunc(projectPach.P00)
	projectPach.P01 = projectFunc(projectPach.P01)
	projectPach.P10 = projectFunc(projectPach.P10)
	projectPach.P11 = projectFunc(projectPach.P11)
	return projectPach
}

func (bp *BilinearPatch) ShouldSplit(lenEdgeMax float32) (canBySplit bool, axis SplitAxis) {
	du0 := bp.P00.Sub(bp.P01).Vec2()
	du1 := bp.P10.Sub(bp.P11).Vec2()
	dv0 := bp.P01.Sub(bp.P11).Vec2()
	dv1 := bp.P00.Sub(bp.P10).Vec2()

	lenU2 := du0.Len() + du1.Len()
	lenV2 := dv0.Len() + dv1.Len()
	maxLen := max(lenU2, lenV2)

	if maxLen < lenEdgeMax {
		return false, SplitAxisNone
	}

	if lenU2 > lenV2 {
		return true, SplitAxisU
	}
	return true, SplitAxisV
}

func (bp *BilinearPatch) SplitByAxis(axis SplitAxis) (BilinearPatch, BilinearPatch) {
	switch axis {
	case SplitAxisV:
		return bp.SubPatch(0.0, 0.5, 0.0, 1.0), bp.SubPatch(0.5, 1.0, 0.0, 1.0)
	case SplitAxisU:
		return bp.SubPatch(0.0, 1.0, 0.0, 0.5), bp.SubPatch(0.0, 1.0, 0.5, 1.0)
	default:
		return *bp, *bp // TODO fix its potetional error
	}
}

func edgeFunction2D(a, b mgl32.Vec3, x, y float32) float32 {
	return (x-a.X())*(b.Y()-a.Y()) -
		(y-a.Y())*(b.X()-a.X())
}

func (bp *BilinearPatch) InsideQuad(sample Sample) bool {
	return edgeFunction2D(bp.P00, bp.P01, sample.X, sample.Y) >= 0 &&
		edgeFunction2D(bp.P01, bp.P11, sample.X, sample.Y) >= 0 &&
		edgeFunction2D(bp.P11, bp.P10, sample.X, sample.Y) >= 0 &&
		edgeFunction2D(bp.P10, bp.P00, sample.X, sample.Y) >= 0
}

// Evaluate возвращает точку на поверхности патча для заданных параметров u и v.
func (bp *BilinearPatch) EvaluatePos(u, v float32) mgl32.Vec3 {
	return EvaluateBilinearVec3(bp.P00, bp.P10, bp.P01, bp.P11, u, v)
}

func (bp *BilinearPatch) EvaluateUV(u, v float32) mgl32.Vec2 {
	return EvaluateBilinearVec2(bp.UV00, bp.UV10, bp.UV01, bp.UV11, u, v)
}

func (bq *BilinearPatch) Dice(dicingRate float32, screenBoundBox BoundBox) (Grid, int, int) {
	width := screenBoundBox.Max.X() - screenBoundBox.Min.X()
	height := screenBoundBox.Max.Y() - screenBoundBox.Min.Y()
	sizeX := int(math.Ceil(float64(width / dicingRate)))
	sizeY := int(math.Ceil(float64(height / dicingRate)))

	grid := Grid{}
	grid.Positions = make([]mgl32.Vec3, 0, (sizeX+1)*(sizeY+1))
	grid.UV = make([]mgl32.Vec2, 0, (sizeX+1)*(sizeY+1))

	// Dice
	for y := 0; y <= sizeY; y++ {
		for x := 0; x <= sizeX; x++ {
			u := float32(x) / float32(sizeX)
			v := float32(y) / float32(sizeY)

			grid.Positions = append(grid.Positions, bq.EvaluatePos(u, v))
			grid.UV = append(grid.UV, bq.EvaluateUV(u, v))
		}
	}

	return grid, sizeX, sizeY
}

func (bp *BilinearPatch) SubPatch(uMin, uMax, vMin, vMax float32) BilinearPatch {
	return BilinearPatch{
		P00:  bp.EvaluatePos(uMin, vMin),
		P01:  bp.EvaluatePos(uMin, vMax),
		P10:  bp.EvaluatePos(uMax, vMin),
		P11:  bp.EvaluatePos(uMax, vMax),
		UV00: bp.EvaluateUV(uMin, vMin),
		UV01: bp.EvaluateUV(uMin, vMax),
		UV10: bp.EvaluateUV(uMax, vMin),
		UV11: bp.EvaluateUV(uMax, vMax),
	}
}

func (bq *BilinearPatch) InverseAffineQuad(sample Sample) (u, v float32) {
	B := bq.P10.Sub(bq.P00)
	C := bq.P01.Sub(bq.P00)
	rhs := mgl32.Vec3{sample.X, sample.Y, 0.0}.Sub(bq.P00)
	det := B.X()*C.Y() - B.Y()*C.X()

	if math.Abs(float64(det)) < 1e-6 {
		return 0, 0
	}
	u = (rhs.X()*C.Y() - rhs.Y()*C.X()) / det
	v = (B.X()*rhs.Y() - B.Y()*rhs.X()) / det

	return u, v
}

func (bp *BilinearPatch) ToBoundBox() (boundBox BoundBox) {
	boundBox.Min = mgl32.Vec3{
		min(bp.P00.X(), bp.P01.X(), bp.P10.X(), bp.P11.X()),
		min(bp.P00.Y(), bp.P01.Y(), bp.P10.Y(), bp.P11.Y()),
		min(bp.P00.Z(), bp.P01.Z(), bp.P10.Z(), bp.P11.Z()),
	}
	boundBox.Max = mgl32.Vec3{
		max(bp.P00.X(), bp.P01.X(), bp.P10.X(), bp.P11.X()),
		max(bp.P00.Y(), bp.P01.Y(), bp.P10.Y(), bp.P11.Y()),
		max(bp.P00.Z(), bp.P01.Z(), bp.P10.Z(), bp.P11.Z()),
	}

	return boundBox
}
