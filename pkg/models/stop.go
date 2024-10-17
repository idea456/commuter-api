package models

type Stop struct {
	Name       string
	Coordinate Coordinate
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}
