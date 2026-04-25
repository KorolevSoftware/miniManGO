package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"

	"github.com/go-gl/mathgl/mgl32"
)

type Render struct {
	Framebuffer      image.Image
	buckets          []*Bucket
	bucketDim        int
	projectionMatrix mgl32.Mat4
	projectToScreen  func(mgl32.Vec3) mgl32.Vec3
}

func (render *Render) SetProjectionMatrix(matrix mgl32.Mat4) {
	render.projectionMatrix = matrix
}

func NewRender(width, height, bucketSize int) *Render {
	render := &Render{}
	render.Framebuffer = image.NewRGBA(image.Rect(0, 0, width, height))
	bucketCountX := width / bucketSize
	bucketCountY := height / bucketSize
	render.bucketDim = bucketSize
	bucketCount := bucketCountX * bucketCountY
	render.buckets = make([]*Bucket, bucketCount)
	render.projectToScreen = func(v mgl32.Vec3) mgl32.Vec3 {
		return Project(v, mgl32.Ident4(), render.projectionMatrix, 0, 0, width, height)
	}
	render.projectionMatrix = mgl32.Ident4()

	for x := range bucketCountX {
		for y := range bucketCountY {
			bucketIndex := x + y*bucketCountX
			startX := x * bucketSize
			startY := y * bucketSize
			render.buckets[bucketIndex] = NewBacket(startX, startY, bucketSize, bucketSize, render.Framebuffer)
		}
	}
	return render
}

func (render *Render) SplitBybucket(bucket *Bucket, patches []BilinearPatch) {
	bucketBB := bucket.toBoundBox()
	for _, bigPatch := range patches {
		bigPatchProjected := bigPatch.Project(render.projectToScreen)
		bigPatchBBox := bigPatchProjected.ToBoundBox()
		if !bucketBB.Intersects(bigPatchBBox) {
			continue
		}

		patchStack := Stack[BilinearPatch]{}
		patchStack.Push(bigPatch)

		for patchStack.Size() > 0 {
			patchToRaster, _ := patchStack.Pop()
			patchToRasterProjected := patchToRaster.Project(render.projectToScreen)
			patchToRasterBBox := patchToRasterProjected.ToBoundBox()
			if !bucketBB.Intersects(patchToRasterBBox) {
				continue
			}
			canBeSplit, axis := patchToRasterProjected.ShouldSplit(10)

			if canBeSplit {
				newPatch1, newPatch2 := patchToRaster.SplitByAxis(axis)
				patchStack.Push(newPatch1)
				patchStack.Push(newPatch2)
				continue
			}

			bucket.AddPrimitive(patchToRaster)
		}
	}
}

func (render *Render) Draw(patches []BilinearPatch, dicingRate float32) {
	fmt.Printf("All patches len: %d\n", len(patches))

	var wg sync.WaitGroup
	for _, bucket := range render.buckets {
		wg.Add(1)
		go func(b *Bucket) {
			defer wg.Done()
			render.SplitBybucket(b, patches)
			b.Draw(dicingRate, render.projectToScreen)
		}(bucket)
	}
	wg.Wait()
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
