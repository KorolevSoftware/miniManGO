package main

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
)

func BenchmarkBilinearPatchDice(b *testing.B) {
	fovyRad := mgl32.DegToRad(60)
	proj := PerspectiveZO(fovyRad, float32(800)/float32(600), 0.1, 30)
	projFunc := func(v mgl32.Vec3) mgl32.Vec3 {
		return Project(v, mgl32.Ident4(), proj, 0, 0, 800, 600)
	}
	patch := BilinearPatch{
		CornerP00: mgl32.Vec3{0, 0, 3},
		CornerP01: mgl32.Vec3{0, 10, 3},
		CornerP10: mgl32.Vec3{10, 0, 3},
		CornerP11: mgl32.Vec3{10, 10, 3},
	}
	ppatch := patch.Project(projFunc)
	box := ppatch.ToBoundBox()
	grid := NewGrid(100, 100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x, y := ppatch.Dice(grid, 1, box)
		_ = x
		_ = y
		if x == 0 || y == 0 {
			b.Error("ожидается хотя бы 1 треугольник и 1 вершину")
		}
	}

}
