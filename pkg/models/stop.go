package models

type Stop struct {
	Id             []string `json:"stop_id"`
	ElementId      string
	Name           string `json:"name"`
	DisplayName    string `json:"display_name"`
	Coordinate     Coordinate
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	PrefixDuration []int
	SuffixDuration []int
}
