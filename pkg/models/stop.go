package models

type Stop struct {
	Id             []string `json:"stop_id"`
	ElementId      string
	Name           string     `json:"name"`
	DisplayName    string     `json:"display_name"`
	Coordinate     Coordinate `json:"coordinates"`
	Latitude       float64    `json:"latitude"`
	Longitude      float64    `json:"longitude"`
	PrefixDuration []int      `json:"prefix_duration"`
	SuffixDuration []int      `json:"suffix_duration"`
}
