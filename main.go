package main

import (
	"fmt"
	"image"
	"image/color"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	"github.com/go-gl/mathgl/mgl32"
) // Это магический импорт, который регистрирует эндпоинты /debug/pprof/

var rocketTexture image.Image

func SampleNear(img image.Image, u, v float32) color.Color {
	bounds := img.Bounds()
	w, h := float32(bounds.Dx()), float32(bounds.Dy())
	px := int(u * w)
	py := int(v * h)
	return img.At(px, py)
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

func LinearizeDepthZO(depth, near, far float32) float32 {
	// depth: ndc.z в [0,1] после PerspectiveZO
	return (near * far) / (far - depth*(far-near))
}

func NormalizeLinearDepthZO(depth, near, far float32) float32 {
	linearDepth := LinearizeDepthZO(depth, near, far)
	return (linearDepth - near) / (far - near)
}

func main() {
	f, _ := os.Create("mem_profile.out")
	defer f.Close()
	file, err := os.Open("Rocket.png")
	if err != nil {
		fmt.Printf("Ошибка при открытии файла: %v\n", err)
		return
	}
	defer file.Close()

	img, format, err := image.Decode(file)
	rocketTexture = img
	if err != nil {
		fmt.Printf("Ошибка при декодировании: %v\n", err)
		return
	}
	fmt.Printf("Формат изображения: %s\n", format)

	bounds := img.Bounds()
	texWidth := bounds.Dx()
	texHeight := bounds.Dy()

	fmt.Printf("Размер: %d x %d\n", texWidth, texHeight)

	model, err := LoadObj("models/cube.obj")

	width, height := 1920, 1080

	fovyRad := mgl32.DegToRad(60)
	proj := PerspectiveZO(fovyRad, float32(width)/float32(height), 0.1, 30)

	render := NewRender(width, height, 32)
	render.SetProjectionMatrix(proj)
	render.Draw(model.Patches, 5)

	render.save("render_image/render.png")
	pprof.WriteHeapProfile(f)
}
