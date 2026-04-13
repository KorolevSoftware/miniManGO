package main

import (
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/go-gl/mathgl/mgl32" // Импорт библиотеки
)

type BilinearPatch struct {
	P00  mgl32.Vec3
	P01  mgl32.Vec3
	P10  mgl32.Vec3
	P11  mgl32.Vec3
	U, V float32
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
func (bp *BilinearPatch) Evaluate(u, v float32) mgl32.Vec3 {
	// Формула билинейной интерполяции:
	// P(u,v) = (1-u)(1-v)P00 + u(1-v)P10 + (1-u)vP01 + uvP11

	term00 := bp.P00.Mul(1.0 - u).Mul(1.0 - v)
	term10 := bp.P10.Mul(u).Mul(1.0 - v)
	term01 := bp.P01.Mul(1.0 - u).Mul(v)
	term11 := bp.P11.Mul(u).Mul(v)

	return term00.Add(term10).Add(term01).Add(term11)
}

func (bound *BoundBox) Contains(point Sample) bool {
	return point.X >= bound.Min.X() && point.X <= bound.Max.X() &&
		point.Y >= bound.Min.Y() && point.Y <= bound.Max.Y() &&
		point.Z >= bound.Min.Z() && point.Z <= bound.Max.Z()
}

func (bp *BilinearPatch) Split(level int, u, v float32) []BilinearPatch {
	if level <= 0 {
		return []BilinearPatch{*bp}
	}

	halfU := u / 2.0
	halfV := v / 2.0

	midCenter := bp.Evaluate(0.5, 0.5)
	midP00_10 := bp.Evaluate(0.5, 0.0)
	midP00_01 := bp.Evaluate(0.0, 0.5)
	midP01_11 := bp.Evaluate(0.5, 1.0)
	midP10_11 := bp.Evaluate(1.0, 0.5)

	// stepSize := 1.0 / float32(level)
	// for x := 0; x < level; x++ {
	// 	for y := 0; y < level; y++ {
	// 		point := bp.Evaluate(stepSize*float32(x), stepSize*float32(y))
	// 	}
	// }

	subPatches := []BilinearPatch{
		{P00: bp.P00, P01: midP00_01, P11: midCenter, P10: midP00_10, U: u - halfU, V: v - halfV},
		{P00: midP00_01, P01: bp.P01, P11: midP01_11, P10: midCenter, U: u + halfU, V: v - halfV},
		{P00: midP00_10, P01: midCenter, P10: bp.P10, P11: midP10_11, U: u - halfU, V: v + halfV},
		{P00: midCenter, P01: midP01_11, P10: midP10_11, P11: bp.P11, U: u + halfU, V: v + halfV},
	}

	if level == 1 {
		return subPatches
	}

	var resultSubPatches []BilinearPatch
	for _, patch := range subPatches {
		childPatches := patch.Split(level-1, halfU, halfV)
		resultSubPatches = append(resultSubPatches, childPatches...)
	}

	return resultSubPatches
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
	bp.P01 = mgl32.Vec3{0.0, 100.0, 0.0}
	bp.P11 = mgl32.Vec3{100.0, 100.0, 0.0}
	bp.P10 = mgl32.Vec3{200.0, 0.0, 0.0}

	mesh := bp.Split(8, 1.0, 1.0)

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
			// Здесь будет ваша математика (Raycasting, BilinearPatch и т.д.)
			// Для примера: сделаем градиент
			//
			sample := Sample{float32(x), float32(y), 0.0}
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
			pixelColor := img.At(int(patch.V*float32(twidth)), int(patch.U*float32(theight)))
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
