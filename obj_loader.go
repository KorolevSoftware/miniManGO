package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/go-gl/mathgl/mgl32"
)

func LoadObj(path string) (*Model, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	model := Model{}
	model.Patches = make([]BilinearPatch, 0, 100)

	var vertexPositions []mgl32.Vec3
	var uvPositions []mgl32.Vec2

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(fields[1], 32)
			y, _ := strconv.ParseFloat(fields[2], 32)
			z, _ := strconv.ParseFloat(fields[3], 32)
			vertexPositions = append(vertexPositions, mgl32.Vec3{float32(x), float32(y), float32(z)})

		case "vt":
			if len(fields) < 3 {
				continue
			}
			x, _ := strconv.ParseFloat(fields[1], 32)
			y, _ := strconv.ParseFloat(fields[2], 32)
			uvPositions = append(uvPositions, mgl32.Vec2{float32(x), float32(1.0 - y)})

		case "f":
			if len(fields) < 5 {
				continue
			}

			var vIndexArr [4]int
			var uvIndexArr [4]int
			for index, pack := range fields[1:] {
				atrIndexesStr := strings.Split(pack, "/")

				vIndex, _ := strconv.ParseInt(atrIndexesStr[0], 10, 32)
				// nIndex, _ := strconv.ParseInt(atrIndexesStr[1], 10, 32)
				uvIndex, _ := strconv.ParseInt(atrIndexesStr[1], 10, 32)
				//
				vIndexArr[index] = int(vIndex - 1)
				uvIndexArr[index] = int(uvIndex - 1)
			}

			patch := BilinearPatch{
				CornerP00: vertexPositions[vIndexArr[0]],
				CornerP01: vertexPositions[vIndexArr[1]],
				CornerP10: vertexPositions[vIndexArr[3]],
				CornerP11: vertexPositions[vIndexArr[2]],
				UV00:      uvPositions[uvIndexArr[0]],
				UV01:      uvPositions[uvIndexArr[1]],
				UV10:      uvPositions[uvIndexArr[3]],
				UV11:      uvPositions[uvIndexArr[2]],
			}
			model.Patches = append(model.Patches, patch)
		}
	}

	return &model, nil
}
