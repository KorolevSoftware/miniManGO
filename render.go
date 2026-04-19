package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"

	"github.com/go-gl/mathgl/mgl32"
)

type Render struct {
	Framebuffer  image.Image
	backets      []Backet
	backetDim    int
	backetCountX int
	backetCountY int
	projecMatrix mgl32.Mat4
}

func (render *Render) SetProject(matrix mgl32.Mat4) {
	render.projecMatrix = matrix
}

func NewRender(width, height, backetSize int) (render *Render) {
	render = &Render{}
	render.Framebuffer = image.NewRGBA(image.Rect(0, 0, width, height))
	backetCountX := width / backetSize
	backetCountY := height / backetSize
	render.backetDim = backetSize
	backetCount := backetCountX * backetCountY
	render.backetCountX = backetCountX
	render.backetCountY = backetCountY
	render.backets = make([]Backet, backetCount)

	for x := 0; x < backetCountX; x++ {
		for y := 0; y < backetCountY; y++ {
			backetIndex := x + y*backetCountX
			startX := x * backetSize
			startY := y * backetSize
			render.backets[backetIndex].init(startX, startY, backetSize, backetSize, render.Framebuffer)
		}
	}
	render.projecMatrix = mgl32.Ident4()
	return
}

func (render *Render) Draw(patches []BilinearPatch) {
	fbBound := render.Framebuffer.Bounds()
	windowBB := BoundBox{
		Min: mgl32.Vec3{float32(fbBound.Min.X), float32(fbBound.Min.Y), 0},
		Max: mgl32.Vec3{float32(fbBound.Max.X), float32(fbBound.Max.Y), 1.0},
	}
	fmt.Printf("All patches len: %d\n", len(patches))
	for _, patch := range patches {
		bbox := patch.ProjectBBox(render.projecMatrix)
		if !windowBB.Intersects(bbox) {
			continue
		}

		splitedPaches := make([]BilinearPatch, 0, 200)

		patch.Split(render.projecMatrix, &splitedPaches)
		fmt.Printf("Patch splited len: %d\n", len(splitedPaches))
		for _, splitedPatch := range splitedPaches {
			bboxToBacket := splitedPatch.ProjectBBox(render.projecMatrix)
			colorddd := color.RGBA{uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), 255}
			splitedPatch.Color = colorddd
			startBacketX := int(math.Floor(float64(bboxToBacket.Min.X() / float32(render.backetDim))))
			startBacketY := int(math.Floor(float64(bboxToBacket.Min.Y() / float32(render.backetDim))))

			startBacketX = max(startBacketX, 0)
			startBacketY = max(startBacketY, 0)

			endBacketX := int(math.Ceil(float64(bboxToBacket.Max.X() / float32(render.backetDim))))
			endBacketY := int(math.Ceil(float64(bboxToBacket.Max.Y() / float32(render.backetDim))))

			endBacketX = min(endBacketX, render.backetCountX)
			endBacketY = min(endBacketY, render.backetCountY)

			splitedPatch.P00 = Project(splitedPatch.P00, mgl32.Ident4(), render.projecMatrix, 0, 0, 800, 608)
			splitedPatch.P01 = Project(splitedPatch.P01, mgl32.Ident4(), render.projecMatrix, 0, 0, 800, 608)
			splitedPatch.P10 = Project(splitedPatch.P10, mgl32.Ident4(), render.projecMatrix, 0, 0, 800, 608)
			splitedPatch.P11 = Project(splitedPatch.P11, mgl32.Ident4(), render.projecMatrix, 0, 0, 800, 608)

			for backetX := startBacketX; backetX < endBacketX; backetX++ {
				for backetY := startBacketY; backetY < endBacketY; backetY++ {

					render.backets[backetX+backetY*render.backetCountX].appPrimitive(splitedPatch)
				}
			}
			// for i := range render.backets {
			// 	render.backets[i].appPrimitive(splitedPatch)
			// }
		}
	}
	// render.backets[0].Draw()
	for _, backet := range render.backets {
		backet.Draw()
	}

}

func (render *Render) save(filepath string) {
	// 4. Сохраняем результат в файл
	f, err := os.Create(filepath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = png.Encode(f, render.Framebuffer)
	if err != nil {
		panic(err)
	}
}
