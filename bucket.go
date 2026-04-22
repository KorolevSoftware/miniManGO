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

type Bucket struct {
	zBuffer        []float32
	SizeX, SizeY   int
	StartX, StartY int
	ColorImage     SubImage
	Primitives     []BilinearPatch
}

func (bucket *Bucket) toBoundBox() (bound BoundBox) {
	bound.Min = mgl32.Vec3{float32(bucket.StartX), float32(bucket.StartY), 0}
	bound.Max = mgl32.Vec3{float32(bucket.StartX + bucket.SizeX), float32(bucket.StartY + bucket.SizeY), 1}
	return
}

func (bucket *Bucket) init(startX, startY, sizeX, sizeY int, imageSrc image.Image) {
	bucket.zBuffer = make([]float32, sizeX*sizeY)
	bucket.StartX = startX
	bucket.StartY = startY
	bucket.SizeX = sizeX
	bucket.SizeY = sizeY
	cropRect := image.Rect(startX, startY, startX+sizeX, startY+sizeY)
	if subber, ok := imageSrc.(SubImage); ok {
		// Внутри этого блока компилятор уже "знает",
		// что у переменной subber точно есть метод SubImage.
		bucket.ColorImage = subber.SubImage(cropRect).(SubImage)
	}

	for index := range bucket.zBuffer {
		bucket.zBuffer[index] = 1.0
	}
}

func (bucket *Bucket) appPrimitive(patch BilinearPatch) {
	bucket.Primitives = append(bucket.Primitives, patch)
}

func (bucket *Bucket) Draw(dicingRate float32, projectFunc func(mgl32.Vec3) mgl32.Vec3) {
	if len(bucket.Primitives) > 0 {
		fmt.Printf("bucket len: %d\n", len(bucket.Primitives))
	}

	for _, patch := range bucket.Primitives {
		// bbox := patch.toBoundBox()
		// startX := int(max(math.Floor(float64(bbox.Min.X())), float64(bucket.StartX)))
		// startY := int(max(math.Floor(float64(bbox.Min.Y())), float64(bucket.StartX)))
		// endX := int(min(math.Ceil(float64(bbox.Max.X())), float64(bucket.StartX+bucket.SizeX)))
		// endY := int(min(math.Ceil(float64(bbox.Max.Y())), float64(bucket.StartY+bucket.SizeY)))
		//
		//
		startX := bucket.StartX
		startY := bucket.StartY
		endX := bucket.StartX + bucket.SizeX
		endY := bucket.StartY + bucket.SizeY

		patchScreen := patch.Project(projectFunc)
		patchScreenBB := patchScreen.ToBoundBox()

		grid, Nx, Ny := patch.Dice(1, patchScreenBB)

		flexPatch := BilinearPatch{}
		gridWidth := Nx + 1

		for i := range Nx {
			for j := range Ny {
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
						if !flexPatch.InsideQuad(sample) {
							continue
						}

						uLocal, vLocal := flexPatch.InverseAffineQuad(sample)
						vpos := flexPatch.EvaluatePos(uLocal, vLocal)
						if bucket.zBuffer[zX+yZ*bucket.SizeX] < vpos.Z() {
							continue
						}
						bucket.zBuffer[zX+yZ*bucket.SizeX] = vpos.Z()
						// resultUV := flexPatch.EvaluateUV(uLocal, vLocal)
						// pixelColor := SampleBilinear(rocketTexture, resultUV.X(), resultUV.Y())
						// bucket.ColorImage.Set(x, y, pixelColor)
						bucket.ColorImage.Set(x, y, flexPatch.Color)
					}
				}
			}
		}
	}

	bucket.Primitives = bucket.Primitives[:]
}
