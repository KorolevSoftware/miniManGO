package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // для поддержки PNG

	"github.com/go-gl/mathgl/mgl32"
	// "math"
	"math/rand"
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

func (backet *Backet) toBoundBox() (bound BoundBox) {
	bound.Min = mgl32.Vec3{float32(backet.StartX), float32(backet.StartY), 0}
	bound.Max = mgl32.Vec3{float32(backet.StartX + backet.SizeX), float32(backet.StartY + backet.SizeY), 1}
	return
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

func (backet *Backet) Draw(dicingRate float32, projectFunc func(mgl32.Vec3) mgl32.Vec3) {
	if len(backet.Primitives) > 0 {
		fmt.Printf("Backet len: %d\n", len(backet.Primitives))
	}

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

		patchScreen := patch.Project(projectFunc)
		patchScreenBB := patchScreen.toBoundBox()

		grid, Nx, Ny := patch.Dice(1, patchScreenBB)

		flexPatch := BilinearPatch{}
		gridWidth := Nx + 1

		for i := 0; i < Nx; i++ {
			for j := 0; j < Ny; j++ {
				flexPatch.P00 = grid.Positions[i+j*gridWidth]
				flexPatch.P01 = grid.Positions[i+(j+1)*gridWidth]
				flexPatch.P10 = grid.Positions[(i+1)+j*gridWidth]
				flexPatch.P11 = grid.Positions[(i+1)+(j+1)*gridWidth]

				flexPatch.UV00 = grid.UV[i+j*gridWidth]
				flexPatch.UV01 = grid.UV[i+(j+1)*gridWidth]
				flexPatch.UV10 = grid.UV[(i+1)+j*gridWidth]
				flexPatch.UV11 = grid.UV[(i+1)+(j+1)*gridWidth]

				flexPatch = flexPatch.Project(projectFunc)
				flexPatch.Color = color.RGBA{uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), 255}
				for x, zX := startX, 0; x < endX; x, zX = x+1, zX+1 {
					for y, yZ := startY, 0; y < endY; y, yZ = y+1, yZ+1 {
						sample := Sample{X: float32(x), Y: float32(y), Z: 0}
						if !flexPatch.insideQuad(sample) {
							continue
						}

						uLocal, vLocal := flexPatch.inverseAffineQuad(sample)
						vpos := flexPatch.EvaluatePos(uLocal, vLocal)
						if backet.zBuffer[zX+yZ*backet.SizeX] < vpos.Z() {
							continue
						}
						backet.zBuffer[zX+yZ*backet.SizeX] = vpos.Z()
						// resultUV := flexPatch.EvaluateUV(uLocal, vLocal)
						// pixelColor := SampleBilinear(rocketTexture, resultUV.X(), resultUV.Y())
						// backet.ColorImage.Set(x, y, pixelColor)
						backet.ColorImage.Set(x, y, flexPatch.Color)
					}
				}
			}
		}
	}

	backet.Primitives = backet.Primitives[:]
}
