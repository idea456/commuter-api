package models

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Property struct {
	PropertyId  string     `json:"PropertyId"`
	CellId      string     `json:"CellId"`
	District    string     `json:"district"`
	Name        string     `json:"name"`
	Address     string     `json:"address"`
	Facilities  []string   `json:"facilities"`
	Link        string     `json:"link"`
	RentalRange string     `json:"rentalRange"`
	Type        string     `json:"type"`
	Coordinates Coordinate `json:"coordinates"`
}
