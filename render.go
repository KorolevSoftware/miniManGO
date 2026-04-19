package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
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
	projectFunc  func(mgl32.Vec3) mgl32.Vec3
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
	render.projectFunc = func(v mgl32.Vec3) mgl32.Vec3 {
		return Project(v, mgl32.Ident4(), render.projecMatrix, 0, 0, width, height)
	}
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

func (render *Render) SplitByBacket(backet *Backet, patches []BilinearPatch) {
	backetBB := backet.toBoundBox()
	for _, bigPatch := range patches {
		bigPatchProjected := bigPatch.Project(render.projectFunc)
		bigPatchBBox := bigPatchProjected.toBoundBox()
		if !backetBB.Intersects(bigPatchBBox) {
			continue
		}

		patchStack := Stack[BilinearPatch]{}
		patchStack.Push(bigPatch)

		for patchStack.Size() > 0 {
			patchToRaster, _ := patchStack.Pop()
			patchToRasterProjected := patchToRaster.Project(render.projectFunc)
			patchToRasterBBox := patchToRasterProjected.toBoundBox()
			if !backetBB.Intersects(patchToRasterBBox) {
				continue
			}
			canBySplit, axis := patchToRasterProjected.CanBySplit(10)

			if canBySplit {
				newPatch1, newPatch2 := patchToRaster.SplitByAxis(axis)
				patchStack.Push(newPatch1)
				patchStack.Push(newPatch2)
				continue
			}

			patchToRasterProjected.Color = color.RGBA{uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), 255}
			backet.appPrimitive(patchToRasterProjected)
		}
	}
}

func (render *Render) Draw(patches []BilinearPatch) {
	fmt.Printf("All patches len: %d\n", len(patches))

	for index := range render.backets {
		render.SplitByBacket(&render.backets[index], patches)
	}

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