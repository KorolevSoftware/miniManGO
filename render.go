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
	buckets      []*Bucket
	bucketDim    int
	bucketCountX int
	bucketCountY int
	projecMatrix mgl32.Mat4
	projectFunc  func(mgl32.Vec3) mgl32.Vec3
}

func (render *Render) SetProject(matrix mgl32.Mat4) {
	render.projecMatrix = matrix
}

func NewRender(width, height, bucketSize int) (render *Render) {
	render = &Render{}
	render.Framebuffer = image.NewRGBA(image.Rect(0, 0, width, height))
	bucketCountX := width / bucketSize
	bucketCountY := height / bucketSize
	render.bucketDim = bucketSize
	bucketCount := bucketCountX * bucketCountY
	render.bucketCountX = bucketCountX
	render.bucketCountY = bucketCountY
	render.buckets = make([]*Bucket, bucketCount)
	render.projectFunc = func(v mgl32.Vec3) mgl32.Vec3 {
		return Project(v, mgl32.Ident4(), render.projecMatrix, 0, 0, width, height)
	}
	for x := range bucketCountX {
		for y := range bucketCountY {
			bucketIndex := x + y*bucketCountX
			startX := x * bucketSize
			startY := y * bucketSize
			render.buckets[bucketIndex] = &Bucket{}
			render.buckets[bucketIndex].init(startX, startY, bucketSize, bucketSize, render.Framebuffer)
		}
	}
	render.projecMatrix = mgl32.Ident4()
	return
}

func (render *Render) SplitBybucket(bucket *Bucket, patches []BilinearPatch) {
	bucketBB := bucket.toBoundBox()
	for _, bigPatch := range patches {
		bigPatchProjected := bigPatch.Project(render.projectFunc)
		bigPatchBBox := bigPatchProjected.ToBoundBox()
		if !bucketBB.Intersects(bigPatchBBox) {
			continue
		}

		patchStack := Stack[BilinearPatch]{}
		patchStack.Push(bigPatch)

		for patchStack.Size() > 0 {
			patchToRaster, _ := patchStack.Pop()
			patchToRasterProjected := patchToRaster.Project(render.projectFunc)
			patchToRasterBBox := patchToRasterProjected.ToBoundBox()
			if !bucketBB.Intersects(patchToRasterBBox) {
				continue
			}
			canBySplit, axis := patchToRasterProjected.ShouldSplit(10)

			if canBySplit {
				newPatch1, newPatch2 := patchToRaster.SplitByAxis(axis)
				patchStack.Push(newPatch1)
				patchStack.Push(newPatch2)
				continue
			}

			patchToRasterProjected.Color = color.RGBA{uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), uint8(rand.Int31n(255)), 255}
			bucket.appPrimitive(patchToRaster)
		}
	}
}

func (render *Render) Draw(patches []BilinearPatch, dicingRate float32) {
	fmt.Printf("All patches len: %d\n", len(patches))

	for _, bucket := range render.buckets {
		render.SplitBybucket(bucket, patches)
		bucket.Draw(dicingRate, render.projectFunc)
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
