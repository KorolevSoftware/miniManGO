package main

import (
	"math"

	"github.com/go-gl/mathgl/mgl32"
)

// P(u,v) = (1-u)(1-v)P00 + u(1-v)P10 + (1-u)vP01 + uvP11
func EvaluateBilinear[T any](p00, p10, p01, p11 T, u, v float32, mul func(T, float32) T, add func(T, T) T) T {
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

func EvaluateBilinearVec3(p00, p10, p01, p11 mgl32.Vec3, u, v float32) mgl32.Vec3 {
	w00 := (1 - u) * (1 - v)
	w10 := u * (1 - v)
	w01 := (1 - u) * v
	w11 := u * v
	return mgl32.Vec3{
		p00[0]*w00 + p10[0]*w10 + p01[0]*w01 + p11[0]*w11,
		p00[1]*w00 + p10[1]*w10 + p01[1]*w01 + p11[1]*w11,
		p00[2]*w00 + p10[2]*w10 + p01[2]*w01 + p11[2]*w11,
	}
}

func EvaluateBilinearVec2(p00, p10, p01, p11 mgl32.Vec2, u, v float32) mgl32.Vec2 {
	w00 := (1 - u) * (1 - v)
	w10 := u * (1 - v)
	w01 := (1 - u) * v
	w11 := u * v
	return mgl32.Vec2{
		p00[0]*w00 + p10[0]*w10 + p01[0]*w01 + p11[0]*w11,
		p00[1]*w00 + p10[1]*w10 + p01[1]*w01 + p11[1]*w11,
	}
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
