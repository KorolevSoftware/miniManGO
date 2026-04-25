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
	CornerP00 mgl32.Vec3
	CornerP01 mgl32.Vec3
	CornerP10 mgl32.Vec3
	CornerP11 mgl32.Vec3
	UV00      mgl32.Vec2
	UV01      mgl32.Vec2
	UV10      mgl32.Vec2
	UV11      mgl32.Vec2
	Color00   color.Color
	Color01   color.Color
	Color10   color.Color
	Color11   color.Color
}

func (patch BilinearPatch) Project(projectToScreen func(mgl32.Vec3) mgl32.Vec3) BilinearPatch {
	patch.CornerP00 = projectToScreen(patch.CornerP00)
	patch.CornerP01 = projectToScreen(patch.CornerP01)
	patch.CornerP10 = projectToScreen(patch.CornerP10)
	patch.CornerP11 = projectToScreen(patch.CornerP11)
	return patch
}

func (patch *BilinearPatch) ShouldSplit(lenEdgeMax float32) (shouldSplit bool, axis SplitAxis) {
	edgeLenU0 := patch.CornerP00.Sub(patch.CornerP01).Vec2()
	edgeLenU1 := patch.CornerP10.Sub(patch.CornerP11).Vec2()
	edgeLenV0 := patch.CornerP01.Sub(patch.CornerP11).Vec2()
	edgeLenV1 := patch.CornerP00.Sub(patch.CornerP10).Vec2()

	totalLenU := edgeLenU0.Len() + edgeLenU1.Len()
	totalLenV := edgeLenV0.Len() + edgeLenV1.Len()
	maxLen := max(totalLenU, totalLenV)

	if maxLen < lenEdgeMax {
		return false, SplitAxisNone
	}

	if totalLenU > totalLenV {
		return true, SplitAxisU
	}
	return true, SplitAxisV
}

func (patch *BilinearPatch) SplitByAxis(axis SplitAxis) (BilinearPatch, BilinearPatch) {
	switch axis {
	case SplitAxisV:
		return patch.SubPatch(0.0, 0.5, 0.0, 1.0), patch.SubPatch(0.5, 1.0, 0.0, 1.0)
	case SplitAxisU:
		return patch.SubPatch(0.0, 1.0, 0.0, 0.5), patch.SubPatch(0.0, 1.0, 0.5, 1.0)
	default:
		return *patch, *patch // TODO fix its potetional error
	}
}

func edgeFunction2D(a, b mgl32.Vec3, x, y float32) float32 {
	return (x-a.X())*(b.Y()-a.Y()) -
		(y-a.Y())*(b.X()-a.X())
}

func (patch *BilinearPatch) InsideQuad(sample Sample) bool {
	return edgeFunction2D(patch.CornerP00, patch.CornerP01, sample.X, sample.Y) >= 0 &&
		edgeFunction2D(patch.CornerP01, patch.CornerP11, sample.X, sample.Y) >= 0 &&
		edgeFunction2D(patch.CornerP11, patch.CornerP10, sample.X, sample.Y) >= 0 &&
		edgeFunction2D(patch.CornerP10, patch.CornerP00, sample.X, sample.Y) >= 0
}

// Evaluate возвращает точку на поверхности патча для заданных параметров u и v.
func (patch *BilinearPatch) EvaluatePos(u, v float32) mgl32.Vec3 {
	return EvaluateBilinearVec3(patch.CornerP00, patch.CornerP10, patch.CornerP01, patch.CornerP11, u, v)
}

func (patch *BilinearPatch) EvaluateUV(u, v float32) mgl32.Vec2 {
	return EvaluateBilinearVec2(patch.UV00, patch.UV10, patch.UV01, patch.UV11, u, v)
}

func (patch *BilinearPatch) Dice(dicingRate float32, screenBoundBox BoundBox) (Grid, int, int) {
	width := screenBoundBox.Max.X() - screenBoundBox.Min.X()
	height := screenBoundBox.Max.Y() - screenBoundBox.Min.Y()
	sizeX := int(math.Ceil(float64(width / dicingRate)))
	sizeY := int(math.Ceil(float64(height / dicingRate)))

	total := (sizeX + 1) * (sizeY + 1)
	grid := Grid{}
	grid.Positions = make([]mgl32.Vec3, total)
	grid.UV = make([]mgl32.Vec2, total)

	idx := 0
	// Dice
	for y := 0; y <= sizeY; y++ {
		for x := 0; x <= sizeX; x++ {
			u := float32(x) / float32(sizeX)
			v := float32(y) / float32(sizeY)

			grid.Positions[idx] = patch.EvaluatePos(u, v)
			grid.UV[idx] = patch.EvaluateUV(u, v)
			idx++
		}
	}

	return grid, sizeX, sizeY
}

func (patch *BilinearPatch) SubPatch(uMin, uMax, vMin, vMax float32) BilinearPatch {
	return BilinearPatch{
		CornerP00: patch.EvaluatePos(uMin, vMin),
		CornerP01: patch.EvaluatePos(uMin, vMax),
		CornerP10: patch.EvaluatePos(uMax, vMin),
		CornerP11: patch.EvaluatePos(uMax, vMax),
		UV00:      patch.EvaluateUV(uMin, vMin),
		UV01:      patch.EvaluateUV(uMin, vMax),
		UV10:      patch.EvaluateUV(uMax, vMin),
		UV11:      patch.EvaluateUV(uMax, vMax),
	}
}

func (patch *BilinearPatch) UnprojectToUV(sample Sample) (u, v float32) {
	B := patch.CornerP10.Sub(patch.CornerP00)
	C := patch.CornerP01.Sub(patch.CornerP00)
	rhs := mgl32.Vec3{sample.X, sample.Y, 0.0}.Sub(patch.CornerP00)
	det := B.X()*C.Y() - B.Y()*C.X()

	if math.Abs(float64(det)) < 1e-6 {
		return 0, 0
	}
	u = (rhs.X()*C.Y() - rhs.Y()*C.X()) / det
	v = (B.X()*rhs.Y() - B.Y()*rhs.X()) / det

	return u, v
}

func (patch *BilinearPatch) ToBoundBox() (boundBox BoundBox) {
	boundBox.Min = mgl32.Vec3{
		min(patch.CornerP00.X(), patch.CornerP01.X(), patch.CornerP10.X(), patch.CornerP11.X()),
		min(patch.CornerP00.Y(), patch.CornerP01.Y(), patch.CornerP10.Y(), patch.CornerP11.Y()),
		min(patch.CornerP00.Z(), patch.CornerP01.Z(), patch.CornerP10.Z(), patch.CornerP11.Z()),
	}
	boundBox.Max = mgl32.Vec3{
		max(patch.CornerP00.X(), patch.CornerP01.X(), patch.CornerP10.X(), patch.CornerP11.X()),
		max(patch.CornerP00.Y(), patch.CornerP01.Y(), patch.CornerP10.Y(), patch.CornerP11.Y()),
		max(patch.CornerP00.Z(), patch.CornerP01.Z(), patch.CornerP10.Z(), patch.CornerP11.Z()),
	}

	return boundBox
}

// GetColorAt проверяет, находится ли точка внутри патча (используя edge functions для квада),
// и если да, возвращает интерполированный цвет через разбиение на треугольники.
func (patch *BilinearPatch) GetColorAt(sample Sample) (color.Color, bool) {
	x, y := sample.X, sample.Y

	// Проверяем попадание в квад, проверяя положение относительно всех 4-х ребер.
	// Для выпуклого квада точка находится внутри, если она лежит по одну сторону от всех ребер.
	// Мы используем edgeFunction2D, которая возвращает знак (положительный или отрицательный)
	// в зависимости от того, с какой стороны от вектора AB находится точка.

	// edgeFunction2D(patch.CornerP00, patch.CornerP01, sample.X, sample.Y) >= 0 &&
	// 	edgeFunction2D(patch.CornerP01, patch.CornerP11, sample.X, sample.Y) >= 0 &&
	// 	edgeFunction2D(patch.CornerP11, patch.CornerP10, sample.X, sample.Y) >= 0 &&
	// 	edgeFunction2D(patch.CornerP10, patch.CornerP00, sample.X, sample.Y) >= 0

	e1 := edgeFunction2D(patch.CornerP00, patch.CornerP01, x, y)
	e2 := edgeFunction2D(patch.CornerP01, patch.CornerP11, x, y)
	e3 := edgeFunction2D(patch.CornerP11, patch.CornerP10, x, y)
	e4 := edgeFunction2D(patch.CornerP10, patch.CornerP00, x, y)

	// Проверяем, что все значения имеют один и тот тот же знак (все >= 0 или все <= 0)
	// Это позволяет корректно работать с любым порядком обхода вершин (CW или CCW).
	isInside := (e1 >= 0 && e2 >= 0 && e3 >= 0 && e4 >= 0) || (e1 <= 0 && e2 <= 0 && e3 <= 0 && e4 <= 0)

	if !isInside {
		return nil, false
	}

	// Разбили на треугольники:
	// T1: P00, P10, P11 (соответствует весам Color00, Color10, Color11)
	// T2: P00, P11, P01 (соответствует весам Color00, Color11, Color01)

	// Проверяем попадание в первый треугольник T1.
	// Используем тот же принцип edge function для проверки внутри треугольника.
	t1_e1 := edgeFunction2D(patch.CornerP00, patch.CornerP01, x, y)
	t1_e2 := edgeFunction2D(patch.CornerP01, patch.CornerP11, x, y)
	t1_e3 := edgeFunction2D(patch.CornerP11, patch.CornerP00, x, y)

	// Если точка в T1 (проверяем знак относительно первого ребра, так как мы уже знаем порядок обхода квада)
	// Но для надежности проверим на совпадение знаков.
	inT1 := (t1_e1 >= 0 && t1_e2 >= 0 && t1_e3 >= 0) || (t1_e1 <= 0 && t1_e2 <= 0 && t1_e3 <= 0)

	if inT1 {
		w0, w1, w2 := patch.getBarycentricCoords(x, y, patch.CornerP00, patch.CornerP01, patch.CornerP11)
		return interpolateColor(patch.Color00, patch.Color01, patch.Color11, w0, w1, w2), true
	}

	// Если не в T1, то она в T2 (так как InsideQuad вернул true)
	w0, w1, w2 := patch.getBarycentricCoords(x, y, patch.CornerP00, patch.CornerP11, patch.CornerP10)
	return interpolateColor(patch.Color00, patch.Color11, patch.Color11, w0, w1, w2), true
}

func (patch *BilinearPatch) getBarycentricCoords(x, y float32, a, b, c mgl32.Vec3) (w0, w1, w2 float32) {
	det := (b.Y()-c.Y())*(a.X()-c.X()) + (c.X()-b.X())*(a.Y()-c.Y())
	// Защита от деления на ноль для вырожденных треугольников
	if math.Abs(float64(det)) < 1e-9 {
		return 1, 0, 0
	}
	w0 = ((b.Y()-c.Y())*(x-c.X()) + (c.X()-b.X())*(y-c.Y())) / det
	w1 = ((c.Y()-a.Y())*(x-c.X()) + (a.X()-c.X())*(y-c.Y())) / det
	w2 = 1.0 - w0 - w1
	return w0, w1, w2
}

func interpolateColor(c0, c1, c2 color.Color, w0, w1, w2 float32) color.Color {
	r0, g0, b0, a0 := c0.RGBA()
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	r := float64(r0)*float64(w0) + float64(r1)*float64(w1) + float64(r2)*float64(w2)
	g := float64(g0)*float64(w0) + float64(g1)*float64(w1) + float64(g2)*float64(w2)
	b := float64(b0)*float64(w0) + float64(b1)*float64(w1) + float64(b2)*float64(w2)
	a := float64(a0)*float64(w0) + float64(a1)*float64(w1) + float64(a2)*float64(w2)

	return color.RGBA64{
		R: uint16(r),
		G: uint16(g),
		B: uint16(b),
		A: uint16(a),
	}
}
