package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/idea456/commuter-api/pkg/services"
)

type Slug struct{}

type GetWalkablePropertiesRequest struct {
	Origin   models.Coordinate `json:"origin"`
	MinPrice float64           `json:"min_price"`
	MaxPrice float64           `json:"max_price"`
	Radius   float64           `json:"radius"`
}

type GetTransitablePropertiesRequest struct {
	Origin   models.Coordinate `json:"origin"`
	MinPrice float64           `json:"min_price"`
	MaxPrice float64           `json:"max_price"`
	Radius   float64           `json:"radius"`
}

type PropertiesController struct {
	propertySvc *services.PropertyService
}

func NewPropertiesController() *PropertiesController {
	return &PropertiesController{
		propertySvc: services.NewPropertyService(),
	}
}

func (ctrl *PropertiesController) GetProperty(w http.ResponseWriter, r *http.Request) {
	fields := r.Context().Value(Slug{}).([]string)
	slug := fields[0]
	w.Write([]byte(slug))
}

func (ctl *PropertiesController) GetWalkableProperties(w http.ResponseWriter, r *http.Request) {
	var body GetWalkablePropertiesRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Invalid argument", http.StatusBadRequest)
		return
	}

	properties, err := ctl.propertySvc.FindWalkablePropertiesByOrigin(r.Context(), body.Origin)
	if err != nil {
		http.Error(w, "TODO", http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(properties)
}

func (ctl *PropertiesController) GetTransitableProperties(w http.ResponseWriter, r *http.Request) {
	latitudeStr := r.URL.Query().Get("latitude")
	longitudeStr := r.URL.Query().Get("longitude")
	if latitudeStr == "" || longitudeStr == "" {
		http.Error(w, "Please specify latitude and longitude.", http.StatusBadRequest)
		return
	}

	latitude, err := strconv.ParseFloat(latitudeStr, 64)
	if err != nil {
		http.Error(w, "latitude must be a float", http.StatusBadRequest)
		return
	}
	longitude, err := strconv.ParseFloat(longitudeStr, 64)
	if err != nil {
		http.Error(w, "longitude must be a float", http.StatusBadRequest)
		return
	}

	minStops := 2
	minStopsStr := r.URL.Query().Get("min_transfer")
	if minStopsStr != "" {
		minStopsQuery, err := strconv.Atoi(minStopsStr)
		if err != nil {
			http.Error(w, "min_stops must be a number", http.StatusBadRequest)
			return
		}
		minStops = minStopsQuery
	}
	maxStops := 2
	maxStopsStr := r.URL.Query().Get("max_transfer")
	if maxStopsStr != "" {
		maxStopsQuery, err := strconv.Atoi(maxStopsStr)
		if err != nil {
			http.Error(w, "max_transfer must be a number", http.StatusBadRequest)
			return
		}
		maxStops = maxStopsQuery
	}

	maxWalkDistance := 1000
	maxWalkDistanceStr := r.URL.Query().Get("walk_distance")
	if maxWalkDistanceStr != "" {
		maxWalkDistanceQuery, err := strconv.Atoi(maxWalkDistanceStr)
		if err != nil {
			http.Error(w, "walk_distance must be a number in metres", http.StatusBadRequest)
			return
		}
		maxWalkDistance = maxWalkDistanceQuery
	}

	origin := models.Coordinate{
		Latitude:  latitude,
		Longitude: longitude,
	}
	properties, err := ctl.propertySvc.FindTransitableProperties(r.Context(), services.FindTransitablePropertiesOptions{
		Range: services.TransferRange{
			MinTransfer: minStops,
			MaxTranfer:  maxStops,
		},
		Origin:       origin,
		WalkDistance: maxWalkDistance,
	})
	if err != nil {
		http.Error(w, "TODO", http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(properties)
}
