package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/go-gl/mathgl/mgl32" // Импорт библиотеки
)

type BilinearPatch struct {
	P00                    mgl32.Vec3
	P01                    mgl32.Vec3
	P10                    mgl32.Vec3
	P11                    mgl32.Vec3
	minU, minV, maxU, maxV float32
}

type BoundBox struct {
	Min mgl32.Vec3
	Max mgl32.Vec3
}

type Sample struct {
	X, Y, Z float32
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
func EvaluateBilinear(p00, p10, p01, p11 mgl32.Vec3, u, v float32) mgl32.Vec3 {
	// Формула билинейной интерполяции:
	// P(u,v) = (1-u)(1-v)P00 + u(1-v)P10 + (1-u)vP01 + uvP11

	term00 := p00.Mul(1.0 - u).Mul(1.0 - v)
	term10 := p10.Mul(u).Mul(1.0 - v)
	term01 := p01.Mul(1.0 - u).Mul(v)
	term11 := p11.Mul(u).Mul(v)

	return term00.Add(term10).Add(term01).Add(term11)
}

// Evaluate возвращает точку на поверхности патча для заданных параметров u и v.
func (bp *BilinearPatch) Evaluate(u, v float32) mgl32.Vec3 {
	return EvaluateBilinear(bp.P00, bp.P10, bp.P01, bp.P11, u, v)
}

func (bound *BoundBox) Contains(point Sample) bool {
	return point.X >= bound.Min.X() && point.X <= bound.Max.X() &&
		point.Y >= bound.Min.Y() && point.Y <= bound.Max.Y() &&
		point.Z >= bound.Min.Z() && point.Z <= bound.Max.Z()
}

func (bp *BilinearPatch) Split(level int, uMin, uMax, vMin, vMax float32) []BilinearPatch {
	if level <= 0 { // Мы возвращаем патч, который был вычислен на основе этих границ
		return []BilinearPatch{{
			P00:  bp.Evaluate(uMin, vMin),
			P01:  bp.Evaluate(uMin, vMax),
			P10:  bp.Evaluate(uMax, vMin),
			P11:  bp.Evaluate(uMax, vMax),
			minU: uMin,
			minV: vMin,
			maxU: uMax,
			maxV: vMax,
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
func SampleBilinear(img image.Image, u, v float32) color.RGBA {
	bounds := img.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())

	x, y := u*w-0.5, v*h-0.5
	x0, y0 := int(x), int(y)
	x1, y1 := x0+1, y0+1

	if x0 < bounds.Min.X {
		x0 = bounds.Min.X
	}
	if y0 < bounds.Min.Y {
		y0 = bounds.Min.Y
	}
	if x1 >= bounds.Max.X {
		x1 = bounds.Max.X - 1
	}
	if y1 >= bounds.Max.Y {
		y1 = bounds.Max.Y - 1
	}

	fx, fy := x-float32(x0), y-float32(y0)

	// Функция для получения float32 цвета [0, 1]
	getF := func(px, py int) (r, g, b, a float32) {
		cr, cg, cb, ca := img.At(px, py).RGBA()
		return float32(cr) / 65535.0, float32(cg) / 65535.0, float32(cb) / 65535.0, float32(ca) / 65535.0
	}

	r00, g00, b00, a00 := getF(x0, y0)
	r10, g10, b10, a10 := getF(x1, y0)
	r01, g01, b01, a01 := getF(x0, y1)
	r11, g11, b11, a11 := getF(x1, y1)

	// Интерполяция X (верх и низ)
	rt := r00*(1-fx) + r10*fx
	gt := g00*(1-fx) + g10*fx
	bt := b00*(1-fx) + b10*fx
	at := a00*(1-fx) + a10*fx

	rb := r01*(1-fx) + r11*fx
	gb := g01*(1-fx) + g11*fx
	bb := b01*(1-fx) + b11*fx
	ab := a01*(1-fx) + a11*fx

	// Интерполяция Y и возврат в uint8 (умножаем на 255!)
	return color.RGBA{
		R: uint8((rt*(1-fy) + rb*fy) * 255), // Ой, скобки! Исправил ниже
		G: uint8((gt*(1-fy) + gb*fy) * 255),
		B: uint8((bt*(1-fy) + bb*fy) * 255),
		A: uint8((at*(1-fy) + ab*fy) * 255),
	}
}

func (bq *BilinearPatch) inverseAffineQuad(sample Sample) (u, v float32) {
	B := bq.P10.Sub(bq.P00)
	C := bq.P01.Sub(bq.P00)
	rhs := mgl32.Vec3{sample.X, sample.Y, 0.0}.Sub(bq.P00)
	det := B.X()*C.Y() - B.Y()*C.X()

	if math.Abs(float64(det)) < 1e-10 {
		return 0, 0
	}
	u = (rhs.X()*C.Y() - rhs.Y()*C.X()) / det
	v = (B.X()*rhs.Y() - B.Y()*rhs.X()) / det

	return u, v
}

func main() {
	file, err := os.Open("Roket.png")
	if err != nil {
		fmt.Printf("Ошибка при открытии файла: %v\n", err)
		return
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	if err != nil {
		fmt.Printf("Ошибка при декодировании: %v\n", err)
		return
	}
	fmt.Printf("Формат изображения: %s\n", format)

	_bounds := img.Bounds()
	twidth := _bounds.Dx()
	theight := _bounds.Dy()

	fmt.Printf("Размер: %d x %d\n", twidth, theight)

	bp := BilinearPatch{}
	bp.P00 = mgl32.Vec3{0.0, 0.0, 0.0}
	bp.P01 = mgl32.Vec3{0.0, 500.0, 0.0}
	bp.P11 = mgl32.Vec3{500.0, 500.0, 0.0}
	bp.P10 = mgl32.Vec3{800.0, 200.0, 0.0}

	mesh := bp.Split(6, 0.0, 1.0, 0.0, 1.0)

	bounds := make([]BoundBox, len(mesh))

	for index, patch := range mesh {
		bounds[index] = patch.toBoundBox()
	}

	// 1. Параметры изображения
	width, height := 800, 600

	// 2. Создаем "холст" (RGBA — это массив пикселей)
	// image.NewRGBA выделяет память под все пиксели сразу
	render := image.NewRGBA(image.Rect(0, 0, width, height))
	fmt.Printf("bounds len: %d\n", len(bounds))
	// 3. Ваш цикл рендеринга (имитация)
	var patch BilinearPatch
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fmt.Printf("progress %f\n", float32(y)/float32(height)*100.0)
			sample := Sample{float32(x) + 0.5, float32(y) + 0.5, 0.0}
			hesHit := false

			for index, bound := range bounds {
				if bound.Contains(sample) && mesh[index].insideQuad(sample) {
					patch = mesh[index]
					hesHit = true
					break
				}
			}

			if !hesHit {
				continue
			}

			uLocal, vLocal := patch.inverseAffineQuad(sample)
			resultUV := EvaluateBilinear(
				mgl32.Vec3{patch.minU, patch.minV, 0},
				mgl32.Vec3{patch.maxU, patch.minV, 0},
				mgl32.Vec3{patch.minU, patch.maxV, 0},
				mgl32.Vec3{patch.maxU, patch.maxV, 0},
				uLocal, vLocal,
			)
			pixelColor := SampleBilinear(img, resultUV.X(), resultUV.Y()) // img.At(int(patch.U*float32(twidth)), int(patch.V*float32(theight)))
			// Устанавливаем цвет пикселя
			render.Set(x, y, pixelColor)
		}
	}

	// 4. Сохраняем результат в файл
	f, err := os.Create("render.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = png.Encode(f, render)
	if err != nil {
		panic(err)
	}
}
