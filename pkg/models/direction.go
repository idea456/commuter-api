package models

import "time"

type Direction struct{}

type PlanDataResponse struct {
	Itineraries []Itineary `json:"itineraries"`
}

type PlanResponse struct {
	Plan PlanDataResponse `json:"plan"`
}

type LegStart struct {
	ScheduledTime string `json:"scheduledTime"`
	Estimated     string `json:"estimated"`
}

type LegEnd struct {
	ScheduledTime string `json:"scheduledTime"`
	Estimated     string `json:"estimated"`
}

type LegGeometry struct {
	Length int    `json:"length"`
	Points string `json:"points"`
}

type LegPoint struct {
	Name string `json:"name"`
}

type Route struct {
	LongName  string `json:"longName"`
	ShortName string `json:"shortName"`
	Color     string `json:"color"`
}
type Leg struct {
	Start       LegStart    `json:"start"`
	End         LegEnd      `json:"end"`
	Mode        string      `json:"mode"`
	Duration    float32     `json:"duration"`
	From        LegPoint    `json:"from"`
	To          LegPoint    `json:"to"`
	LegGeometry LegGeometry `json:"legGeometry"`
	Distance    float64     `json:"distance"`
	Route       Route       `json:"route"`
}

type Itineary struct {
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	Duration     int       `json:"duration"`
	WalkDistance float64   `json:"walkDistance"`
	WalkTime     float64   `json:"walkTime"`
	WaitingTime  int       `json:"waitingTime"`
	Legs         []Leg     `json:"legs"`
}

type DirectionOptions struct {
	WalkReluctance int      `json:"walk_reluctance"`
	TransportModes []string `json:"transport_modes"`
}
