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

func (bp *BilinearPatch) ProjectBBox(matrix mgl32.Mat4) (bound BoundBox) {
	p1 := Project(bp.P00, mgl32.Ident4(), matrix, 0, 0, 800, 608)
	p2 := Project(bp.P01, mgl32.Ident4(), matrix, 0, 0, 800, 608)
	p3 := Project(bp.P10, mgl32.Ident4(), matrix, 0, 0, 800, 608)
	p4 := Project(bp.P11, mgl32.Ident4(), matrix, 0, 0, 800, 608)

	// p1 := matrix.Mul4x1(bp.P00.Vec4(1)).Vec3()
	// p2 := matrix.Mul4x1(bp.P01.Vec4(1)).Vec3()
	// p3 := matrix.Mul4x1(bp.P10.Vec4(1)).Vec3()
	// p4 := matrix.Mul4x1(bp.P11.Vec4(1)).Vec3()

	// matrix.Mul4x1(du0.Vec4(1)).Vec3()

	bound.Min = mgl32.Vec3{
		min(p1.X(), p2.X(), p3.X(), p4.X()),
		min(p1.Y(), p2.Y(), p3.Y(), p4.Y()),
		min(p1.Z(), p2.Z(), p3.Z(), p4.Z()),
	}
	bound.Max = mgl32.Vec3{
		max(p1.X(), p2.X(), p3.X(), p4.X()),
		max(p1.Y(), p2.Y(), p3.Y(), p4.Y()),
		max(p1.Z(), p2.Z(), p3.Z(), p4.Z()),
	}
	return
}

func (bp *BilinearPatch) Split(matrix mgl32.Mat4, splitPaches *[]BilinearPatch) {
	isNeedSplit, axis := bp.CanBySplit(matrix)

	if !isNeedSplit {
		*splitPaches = append(*splitPaches, *bp)
		return
	}

	var p1, p2 BilinearPatch

	switch axis {
	case SplitAxisV:
		p1 = bp.SubPatch(0.0, 0.5, 0.0, 1.0)
		p2 = bp.SubPatch(0.5, 1.0, 0.0, 1.0)
	case SplitAxisU:
		p1 = bp.SubPatch(0.0, 1.0, 0.0, 0.5)
		p2 = bp.SubPatch(0.0, 1.0, 0.5, 1.0)
	}

	p1.Split(matrix, splitPaches)
	p2.Split(matrix, splitPaches)
}

func (bp *BilinearPatch) CanBySplit(matrix mgl32.Mat4) (canBySplit bool, axis SplitAxis) {
	p00 := Project(bp.P00, mgl32.Ident4(), matrix, 0, 0, 800, 608)
	p01 := Project(bp.P01, mgl32.Ident4(), matrix, 0, 0, 800, 608)
	p10 := Project(bp.P10, mgl32.Ident4(), matrix, 0, 0, 800, 608)
	p11 := Project(bp.P11, mgl32.Ident4(), matrix, 0, 0, 800, 608)

	du0 := p00.Sub(p01)
	du1 := p10.Sub(p11)
	dv0 := p01.Sub(p11)
	dv1 := p00.Sub(p10)

	lenU2 := du0.LenSqr() + du1.LenSqr()
	lenV2 := dv0.LenSqr() + dv1.LenSqr()
	maxLen := max(lenU2, lenV2)

	if maxLen < 1000^2 {
		return false, SplitAxisNone
	}

	if lenU2 > lenV2 {
		return true, SplitAxisU
	}
	return true, SplitAxisV
}

func edgeFunction2D(a, b mgl32.Vec3, x, y float32) float32 {
	return (x-a.X())*(b.Y()-a.Y()) -
		(y-a.Y())*(b.X()-a.X())
}

func (bp *BilinearPatch) insideQuad(sample Sample) bool {
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

func (bp *BilinearPatch) SplitQuad(level int, uMin, uMax, vMin, vMax float32) []BilinearPatch {
	if level <= 0 { // Мы возвращаем патч, который был вычислен на основе этих границ
		return []BilinearPatch{{
			P00:  bp.EvaluatePos(uMin, vMin),
			P01:  bp.EvaluatePos(uMin, vMax),
			P10:  bp.EvaluatePos(uMax, vMin),
			P11:  bp.EvaluatePos(uMax, vMax),
			UV00: bp.EvaluateUV(uMin, vMin),
			UV01: bp.EvaluateUV(uMin, vMax),
			UV10: bp.EvaluateUV(uMax, vMin),
			UV11: bp.EvaluateUV(uMax, vMax),
		}}
	}

	var subPatches []BilinearPatch
	midU := (uMin + uMax) / 2.0
	midV := (vMin + vMax) / 2.0

	// Определяем 4 квадранта: {uStart, uEnd, vStart, vEnd}
	quads := [4][4]float32{
		{uMin, midU, vMin, midV}, // BL (P00)
		{uMin, midU, midV, vMax}, // TL (P01)
		{midU, uMax, vMin, midV}, // BR (P10)
		{midU, uMax, midV, vMax}, // TR (P11)
	}

	for _, q := range quads {
		u0, u1, v0, v1 := q[0], q[1], q[2], q[3]

		// ВАЖНО: Мы создаем новый патч, вычисляя его углы через Evaluate
		// используя параметры U и V.
		// Это гарантирует, что геометрия будет идеально соответствовать текстуре!
		// ВАЖНО: Мы создаем новый патч, вычисляя его углы через Evaluate
		// используя параметры U и V.
		// Это гарантирует, что геометрия будет идеально соответствовать текстуре!
		// newPatch := BilinearPatch{
		// 	P00: bp.Evaluate(u0, v0),
		// 	P01: bp.Evaluate(u0, v1),
		// 	P10: bp.Evaluate(u1, v0),
		// 	P11: bp.Evaluate(u1, v1),
		// 	U:   u0, // Сохраняем глобальный U для текстурирования
		// 	V:   v0, // Сохраняем глобальный V для текстурирования
		// }

		// Рекурсия
		subPatches = append(subPatches, bp.SplitQuad(level-1, u0, u1, v0, v1)...)
	}

	return subPatches
}

func (bq *BilinearPatch) inverseAffineQuad(sample Sample) (u, v float32) {
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

func (bp *BilinearPatch) toBoundBox() BoundBox {
	boundBox := BoundBox{}
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
