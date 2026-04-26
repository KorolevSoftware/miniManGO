package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // для поддержки PNG

	"github.com/go-gl/mathgl/mgl32"
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

func NewBacket(startX, startY, sizeX, sizeY int, imageSrc image.Image) *Bucket {
	bucket := &Bucket{}
	bucket.zBuffer = make([]float32, sizeX*sizeY)
	bucket.StartX = startX
	bucket.StartY = startY
	bucket.SizeX = sizeX
	bucket.SizeY = sizeY
	cropRect := image.Rect(startX, startY, startX+sizeX, startY+sizeY)
	if subImage, ok := imageSrc.(SubImage); ok {
		bucket.ColorImage = subImage.SubImage(cropRect).(SubImage)
	}

	for index := range bucket.zBuffer {
		bucket.zBuffer[index] = 1.0
	}
	return bucket
}

func (bucket *Bucket) AddPrimitive(patch BilinearPatch) {
	bucket.Primitives = append(bucket.Primitives, patch)
}

func (bucket *Bucket) Draw(dicingRate float32, projectToScreen func(mgl32.Vec3) mgl32.Vec3) {
	fmt.Printf("bucket len: %d\n", len(bucket.Primitives))
	var micropolygon BilinearPatch

	for _, patch := range bucket.Primitives {

		backetStartX := bucket.StartX
		backetStartY := bucket.StartY
		backetEndX := bucket.StartX + bucket.SizeX
		backetEndY := bucket.StartY + bucket.SizeY

		patchScreen := patch.Project(projectToScreen)
		patchScreenBB := patchScreen.ToBoundBox()

		grid, Nx, Ny := patch.Dice(dicingRate, patchScreenBB)

		gridWidth := Nx + 1

		for i := range Nx {
			for j := range Ny {
				micropolygon.CornerP00 = grid.Positions[i+j*gridWidth]
				micropolygon.CornerP01 = grid.Positions[i+(j+1)*gridWidth]
				micropolygon.CornerP10 = grid.Positions[(i+1)+j*gridWidth]
				micropolygon.CornerP11 = grid.Positions[(i+1)+(j+1)*gridWidth]

				micropolygon.UV00 = grid.UV[i+j*gridWidth]
				micropolygon.UV01 = grid.UV[i+(j+1)*gridWidth]
				micropolygon.UV10 = grid.UV[(i+1)+j*gridWidth]
				micropolygon.UV11 = grid.UV[(i+1)+(j+1)*gridWidth]

				micropolygon = micropolygon.Project(projectToScreen)

				micropolygon.Color00 = SampleNear(rocketTexture, micropolygon.UV00.X(), micropolygon.UV00.Y())
				micropolygon.Color01 = SampleNear(rocketTexture, micropolygon.UV01.X(), micropolygon.UV01.Y())
				micropolygon.Color10 = SampleNear(rocketTexture, micropolygon.UV10.X(), micropolygon.UV10.Y())
				micropolygon.Color11 = SampleNear(rocketTexture, micropolygon.UV11.X(), micropolygon.UV11.Y())

				bbMicropolygon := micropolygon.ToBoundBox()
				bStartX, bStartY, bEndX, bEndY := bbMicropolygon.Int()

				startX := max(backetStartX, bStartX)
				startY := max(backetStartY, bStartY)

				endX := min(backetEndX, bEndX)
				endY := min(backetEndY, bEndY)

				for x := startX; x < endX; x++ {
					for y := startY; y < endY; y++ {
						sample := Sample{X: float32(x), Y: float32(y), Z: 0}
						if !micropolygon.InsideQuad(sample) {
							continue
						}

						uLocal, vLocal := micropolygon.UnprojectToUV(sample)
						vpos := micropolygon.EvaluatePos(uLocal, vLocal)
						zposX := x - backetStartX
						zposY := y - backetStartY

						if bucket.zBuffer[zposX+zposY*bucket.SizeX] < vpos.Z() {
							continue
						}

						bucket.zBuffer[zposX+zposY*bucket.SizeX] = vpos.Z()
						resultUV := micropolygon.EvaluateUV(uLocal, vLocal)
						pixelColor := SampleBilinear(rocketTexture, resultUV.X(), resultUV.Y())
						bucket.ColorImage.Set(x, y, pixelColor)
						// bucket.ColorImage.Set(x, y, micropolygon.Color)
					}
				}
			}
		}
	}
}
