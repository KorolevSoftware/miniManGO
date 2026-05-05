package main

import "github.com/go-gl/mathgl/mgl32"

type Grid struct {
	Positions     []mgl32.Vec3
	UV            []mgl32.Vec2
	Color         []mgl32.Vec3
	width, height int
}

func (grid *Grid) updateGridData(width, height int) {
	grid.width = width
	grid.height = height

	grid.Positions = make([]mgl32.Vec3, width*height)
	grid.Color = make([]mgl32.Vec3, width*height)
	grid.UV = make([]mgl32.Vec2, width*height)
}

func NewGrid(width, height int) *Grid {
	grid := &Grid{}
	grid.updateGridData(width, height)
	return grid
}

func (grid *Grid) SetSize(width, height int) {
	if grid.height >= height && grid.width >= width {
		return
	}  
	grid.updateGridData(width, height)
}
