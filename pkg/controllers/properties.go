package controllers

import (
	"encoding/json"
	"net/http"

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
	var body GetTransitablePropertiesRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, "Invalid argument", http.StatusBadRequest)
		return
	}

	properties, err := ctl.propertySvc.FindTransitableProperties(r.Context(), body.Origin)
	if err != nil {
		http.Error(w, "TODO", http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(properties)
}
