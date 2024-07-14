package utils

import (
	"fmt"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/uber/h3-go/v4"
)

const H3_RESOLUTION = 8

func Test() {
	coor := h3.LatLng{
		Lat: 3.1426108381916618,
		Lng: 101.7168775685601,
	}
	cell := h3.LatLngToCell(coor, 6)

	fmt.Println((cell))

	disk := h3.GridDiskDistances(cell, 5)
	fmt.Println(disk)
}

func ToCell(coor models.Coordinate) *h3.Cell {
	cell := h3.LatLngToCell(h3.LatLng{
		Lat: coor.Latitude,
		Lng: coor.Longitude,
	}, H3_RESOLUTION)

	return &cell
}
