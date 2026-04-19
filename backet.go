package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // для поддержки PNG
	// "math"
	// "math/rand"
)

type SubImage interface {
	SubImage(r image.Rectangle) image.Image
	Set(x, y int, c color.Color)
}

type Backet struct {
	zBuffer        []float32
	SizeX, SizeY   int
	StartX, StartY int
	ColorImage     SubImage
	Primitives     []BilinearPatch
}

func (backet *Backet) init(startX, startY, sizeX, sizeY int, imageSrc image.Image) {
	backet.zBuffer = make([]float32, sizeX*sizeY)
	backet.StartX = startX
	backet.StartY = startY
	backet.SizeX = sizeX
	backet.SizeY = sizeY
	cropRect := image.Rect(startX, startY, startX+sizeX, startY+sizeY)
	if subber, ok := imageSrc.(SubImage); ok {
		// Внутри этого блока компилятор уже "знает",
		// что у переменной subber точно есть метод SubImage.
		backet.ColorImage = subber.SubImage(cropRect).(SubImage)
	}

	for index := range backet.zBuffer {
		backet.zBuffer[index] = 1.0
	}
}

func (backet *Backet) appPrimitive(patch BilinearPatch) {
	backet.Primitives = append(backet.Primitives, patch)
}

func (backet *Backet) Draw() {
	if len(backet.Primitives) > 0 {
		fmt.Printf("Backet len: %d\n", len(backet.Primitives))
	}

	// colorddd := color.RGBA{uint8(rand.Int31n(255)), 255, 0, 255}
	// for x := 0; x < backet.SizeX; x++ {
	// 	for y := 0; y < backet.SizeY; y++ {
	// 		backet.ColorImage.Set(x+backet.StartX, y+backet.StartY, colorddd)
	// 	}
	// }
	// return
	for _, patch := range backet.Primitives {
		// bbox := patch.toBoundBox()
		// startX := int(max(math.Floor(float64(bbox.Min.X())), float64(backet.StartX)))
		// startY := int(max(math.Floor(float64(bbox.Min.Y())), float64(backet.StartX)))
		// endX := int(min(math.Ceil(float64(bbox.Max.X())), float64(backet.StartX+backet.SizeX)))
		// endY := int(min(math.Ceil(float64(bbox.Max.Y())), float64(backet.StartY+backet.SizeY)))
		//
		//
		startX := backet.StartX
		startY := backet.StartY
		endX := backet.StartX + backet.SizeX
		endY := backet.StartY + backet.SizeY

		for x, zX := startX, 0; x < endX; x, zX = x+1, zX+1 {
			for y, yZ := startY, 0; y < endY; y, yZ = y+1, yZ+1 {
				sample := Sample{X: float32(x), Y: float32(y), Z: 0}
				if !patch.insideQuad(sample) {
					continue
				}

				uLocal, vLocal := patch.inverseAffineQuad(sample)
				vpos := patch.EvaluatePos(uLocal, vLocal)
				if backet.zBuffer[zX+yZ*backet.SizeX] < vpos.Z() {
					continue
				}
				backet.zBuffer[zX+yZ*backet.SizeX] = vpos.Z()
				// resultUV := patch.EvaluateUV(uLocal, vLocal)
				// pixelColor := SampleBilinear(rocketTexture, resultUV.X(), resultUV.Y())
				// backet.ColorImage.Set(x, y, color.RGBA{255, 0, 0, 255})
				backet.ColorImage.Set(x, y, patch.Color)
			}
		}
	}

	backet.Primitives = backet.Primitives[:]
}
