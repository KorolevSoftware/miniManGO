package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// EvaluateBilinear — это универсальная функция, которая работает с любым типом T,
// если мы предоставим ей способы умножения и сложения для этого типа.
func EvaluateBilinear[T any](p00, p10, p01, p11 T, u, v float32, mul func(T, float32) T, add func(T, T) T) T {
	// P(u,v) = (1-u)(1-v)P00 + u(1-v)P10 + (1-u)vP01 + uvP11

	// Для оптимизации вычисляем веса один раз
	w00 := (1.0 - u) * (1.0 - v)
	w10 := u * (1.0 - v)
	w01 := (1.0 - u) * v
	w11 := u * v

	term00 := mul(p00, w00)
	term10 := mul(p10, w10)
	term01 := mul(p01, w01)
	term11 := mul(p11, w11)

	return add(add(term00, term10), add(term01, term11))
}

// Специализированная версия для mgl32.Vec3 (чтобы не писать callback-и каждый раз)
func EvaluateBilinearVec3(p00, p10, p01, p11 mgl32.Vec3, u, v float32) mgl32.Vec3 {
	return EvaluateBilinear(p00, p10, p01, p11, u, v,
		func(v mgl32.Vec3, s float32) mgl32.Vec3 { return v.Mul(s) },
		func(v1, v2 mgl32.Vec3) mgl32.Vec3 { return v1.Add(v2) },
	)
}

// Специализированная версия для mgl32.Vec2 (чтобы не писать callback-и каждый раз)
func EvaluateBilinearVec2(p00, p10, p01, p11 mgl32.Vec2, u, v float32) mgl32.Vec2 {
	return EvaluateBilinear(p00, p10, p01, p11, u, v,
		func(v mgl32.Vec2, s float32) mgl32.Vec2 { return v.Mul(s) },
		func(v1, v2 mgl32.Vec2) mgl32.Vec2 { return v1.Add(v2) },
	)
}

// Специализированная версия для float32 (для каналов цвета)
func EvaluateBilinearFloat(p00, p10, p01, p11 float32, u, v float32) float32 {
	return EvaluateBilinear(p00, p10, p01, p11, u, v,
		func(f float32, s float32) float32 { return f * s },
		func(f1, f2 float32) float32 { return f1 + f2 },
	)
}

func PerspectiveZO(fovy, aspect, near, far float32) mgl32.Mat4 {
	f := float32(1.0 / math.Tan(float64(fovy)*0.5))
	nmf := near - far

	return mgl32.Mat4{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, far / nmf, -1,
		0, 0, (far * near) / nmf, 0,
	}
}

func Project(obj mgl32.Vec3, modelview, projection mgl32.Mat4, initialX, initialY, width, height int) (win mgl32.Vec3) {
	obj4 := mgl32.Vec4{obj.X(), obj.Y(), obj.Z(), 1}

	clip := projection.Mul4(modelview).Mul4x1(obj4)
	if clip.W() == 0 {
		return mgl32.Vec3{}
	}

	invW := float32(1.0) / clip.W()
	ndc := clip.Mul(invW)

	// X: [-1,1] -> [initialX, initialX+width]
	win[0] = float32(initialX) + float32(width)*(ndc[0]+1.0)*0.5

	// Y: переворот, чтобы экранный Y рос вниз
	win[1] = float32(initialY) + float32(height)*(1.0-(ndc[1]+1.0)*0.5)
	win[2] = ndc[2] //[0,1]

	return win
}
