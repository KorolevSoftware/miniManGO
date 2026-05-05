package main

import (
	"fmt"
	"image"

	"github.com/go-gl/mathgl/mgl32"
)

type Bucket struct {
	zBuffer        []float32
	SizeX, SizeY   int
	StartX, StartY int
	ColorImage     *image.RGBA
	Primitives     []BilinearPatch
}

func (bucket *Bucket) toBoundBox() (bound BoundBox) {
	bound.Min = mgl32.Vec3{float32(bucket.StartX), float32(bucket.StartY), 0}
	bound.Max = mgl32.Vec3{float32(bucket.StartX + bucket.SizeX), float32(bucket.StartY + bucket.SizeY), 1}
	return
}

func NewBucket(startX, startY, sizeX, sizeY int, imageSrc *image.RGBA) *Bucket {
	bucket := &Bucket{}
	bucket.zBuffer = make([]float32, sizeX*sizeY)
	bucket.StartX = startX
	bucket.StartY = startY
	bucket.SizeX = sizeX
	bucket.SizeY = sizeY
	cropRect := image.Rect(startX, startY, startX+sizeX, startY+sizeY)
	bucket.ColorImage = imageSrc.SubImage(cropRect).(*image.RGBA)

	for i := range bucket.zBuffer {
		bucket.zBuffer[i] = 1.0
	}
	return bucket
}

func (bucket *Bucket) AddPrimitive(patch BilinearPatch) {
	bucket.Primitives = append(bucket.Primitives, patch)
}

func (bucket *Bucket) setPixel(x, y int, color mgl32.Vec4) {
	i := bucket.ColorImage.PixOffset(x, y)
	bucket.ColorImage.Pix[i] = uint8(color[0] * 255)
	bucket.ColorImage.Pix[i+1] = uint8(color[1] * 255)
	bucket.ColorImage.Pix[i+2] = uint8(color[2] * 255)
	bucket.ColorImage.Pix[i+3] = uint8(color[3] * 255)
}

func (bucket *Bucket) Draw(dicingRate float32, projectToScreen func(mgl32.Vec3) mgl32.Vec3) {
	fmt.Printf("bucket len: %d\n", len(bucket.Primitives))
	var micropolygon BilinearPatch
	grid := NewGrid(100, 100)

	for _, patch := range bucket.Primitives {
		bucketStartX := bucket.StartX
		bucketStartY := bucket.StartY
		bucketEndX := bucket.StartX + bucket.SizeX
		bucketEndY := bucket.StartY + bucket.SizeY

		patchScreen := patch.Project(projectToScreen)
		patchScreenBB := patchScreen.ToBoundBox()

		Nx, Ny := patch.Dice(grid, dicingRate, patchScreenBB)
		totalVertex := (Nx + 1) * (Ny + 1)
		for idx := range totalVertex { // Shader
			grid.Color[idx] = ColorShader(grid.Positions[idx], grid.UV[idx])
		}

		for idx := range totalVertex { // Project to Screen
			grid.Positions[idx] = projectToScreen(grid.Positions[idx])
		}

		gridWidth := Nx + 1

		for i := range Nx {
			for j := range Ny {
				micropolygon.CornerP00 = grid.Positions[i+j*gridWidth]
				micropolygon.CornerP01 = grid.Positions[i+(j+1)*gridWidth]
				micropolygon.CornerP10 = grid.Positions[(i+1)+j*gridWidth]
				micropolygon.CornerP11 = grid.Positions[(i+1)+(j+1)*gridWidth]

				micropolygon.Color00 = grid.Color[i+j*gridWidth]
				micropolygon.Color01 = grid.Color[i+(j+1)*gridWidth]
				micropolygon.Color10 = grid.Color[(i+1)+j*gridWidth]
				micropolygon.Color11 = grid.Color[(i+1)+(j+1)*gridWidth]

				bbMicropolygon := micropolygon.ToBoundBox()
				bStartX, bStartY, bEndX, bEndY := bbMicropolygon.Int()

				startX := max(bucketStartX, bStartX)
				startY := max(bucketStartY, bStartY)
				endX := min(bucketEndX, bEndX)
				endY := min(bucketEndY, bEndY)

				for x := startX; x < endX; x++ {
					for y := startY; y < endY; y++ {
						sample := Sample{X: float32(x), Y: float32(y), Z: 0}
						if !micropolygon.InsideQuad(sample) {
							continue
						}
						zposX := x - bucketStartX
						zposY := y - bucketStartY
						uLocal, vLocal := micropolygon.UnprojectToUV(sample)
						vpos := micropolygon.EvaluatePos(uLocal, vLocal)

						if bucket.zBuffer[zposX+zposY*bucket.SizeX] < vpos.Z() {
							continue
						}

						bucket.zBuffer[zposX+zposY*bucket.SizeX] = vpos.Z()
						resultColor := micropolygon.EvaluateColor(uLocal, vLocal)
						bucket.setPixel(x, y, resultColor.Vec4(1.0))
					}
				}
			}
		}
	}
}
