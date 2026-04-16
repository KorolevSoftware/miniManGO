package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

type BilinearPatch struct {
	P00  mgl32.Vec3
	P01  mgl32.Vec3
	P10  mgl32.Vec3
	P11  mgl32.Vec3
	UV00 mgl32.Vec2
	UV01 mgl32.Vec2
	UV10 mgl32.Vec2
	UV11 mgl32.Vec2
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

func (bp *BilinearPatch) Split(level int, uMin, uMax, vMin, vMax float32) []BilinearPatch {
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
		// newPatch := BilinearPatch{
		// 	P00: bp.Evaluate(u0, v0),
		// 	P01: bp.Evaluate(u0, v1),
		// 	P10: bp.Evaluate(u1, v0),
		// 	P11: bp.Evaluate(u1, v1),
		// 	U:   u0, // Сохраняем глобальный U для текстурирования
		// 	V:   v0, // Сохраняем глобальный V для текстурирования
		// }

		// Рекурсия
		subPatches = append(subPatches, bp.Split(level-1, u0, u1, v0, v1)...)
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
