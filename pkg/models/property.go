package models

type Coordinate struct {
	Latitude  float64 `json:"latitude" bson:"latitude"`
	Longitude float64 `json:"longitude" bson:"longitude"`
}

type RentalRange struct {
	FromPrice float64 `json:"fromPrice" bson:"fromPrice"`
	ToPrice   float64 `json:"toPrice" bson:"toPrice"`
}

type Property struct {
	Id          string      `json:"id" bson:"id"`
	CellId      string      `json:"cellId" bson:"cellId"`
	District    string      `json:"district" bson:"district"`
	Name        string      `json:"name" bson:"name"`
	Address     string      `json:"address" bson:"address"`
	Facilities  []string    `json:"facilities" bson:"facilities"`
	Link        string      `json:"link" bson:"link"`
	RentalRange RentalRange `json:"rentalRange" bson:"rentalRange"`
	Type        string      `json:"type" bson:"type"`
	Coordinates Coordinate  `json:"coordinates" bson:"coordinates"`
	Distance    float64     `json:"distance"`
}

type FindNearestPropertiesFilter struct {
	MinPrice float64
	MaxPrice float64
	Radius   float64
}
