package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/go-gl/mathgl/mgl32"
)

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

	model, err := LoadObj("models/cube.obj")
	allPaches := make([]BilinearPatch, 0, 100)

	for _, patch := range model.patches {
		allPaches = append(allPaches, patch.Split(6, 0.0, 1.0, 0.0, 1.0)...)
	}

	// 1. Параметры изображения
	width, height := 800, 600

	fovyRad := mgl32.DegToRad(60)
	proj := PerspectiveZO(fovyRad, float32(width)/float32(height), 0.1, 20)

	for i, patch := range allPaches {
		allPaches[i].P00 = Project(patch.P00, mgl32.Ident4(), proj, 0, 0, width, height)
		allPaches[i].P01 = Project(patch.P01, mgl32.Ident4(), proj, 0, 0, width, height)
		allPaches[i].P10 = Project(patch.P10, mgl32.Ident4(), proj, 0, 0, width, height)
		allPaches[i].P11 = Project(patch.P11, mgl32.Ident4(), proj, 0, 0, width, height)
	}

	bounds := make([]BoundBox, len(allPaches))

	for index, patch := range allPaches {
		bounds[index] = patch.toBoundBox()
	}

	// 2. Создаем "холст" (RGBA — это массив пикселей)
	// image.NewRGBA выделяет память под все пиксели сразу
	render := image.NewRGBA(image.Rect(0, 0, width, height))
	fmt.Printf("bounds len: %d\n", len(bounds))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fmt.Printf("progress %f\n", float32(y)/float32(height)*100.0)
			sample := Sample{float32(x) + 0.5, float32(y) + 0.5, 0.0}

			var depth float32 = 1.0

			for index, bound := range bounds {
				if bound.Contains(sample) && allPaches[index].insideQuad(sample) { //&& allPaches[index].insideQuad(sample)
					patch := allPaches[index]
					uLocal, vLocal := patch.inverseAffineQuad(sample)
					vpos := patch.EvaluatePos(uLocal, vLocal)
					if depth < vpos.Z() {
						continue
					}
					depth = vpos.Z()
					resultUV := patch.EvaluateUV(uLocal, vLocal)
					pixelColor := SampleBilinear(img, resultUV.X(), resultUV.Y())
					render.Set(x, y, pixelColor)

				}
			}
		}
	}

	// 4. Сохраняем результат в файл
	f, err := os.Create("render_image/render.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = png.Encode(f, render)
	if err != nil {
		panic(err)
	}
}
