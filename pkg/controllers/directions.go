package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/idea456/commuter-api/pkg/services"
)

type GetDirectionsBody struct {
	Origin      models.Coordinate       `json:"origin"`
	Destination models.Coordinate       `json:"destination"`
	Options     models.DirectionOptions `json:"options"`
}

type DirectionsController struct {
	directionsService *services.DirectionService
}

func NewDirectionsController() (*DirectionsController, error) {
	directionsSvc, err := services.NewDirectionService()
	if err != nil {
		return nil, err
	}
	return &DirectionsController{
		directionsService: directionsSvc,
	}, nil
}

func (ctl *DirectionsController) GetDirections(w http.ResponseWriter, r *http.Request) {
	var body GetDirectionsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid body format", http.StatusBadRequest)
		return
	}

	plan, err := ctl.directionsService.GetDirections(body.Origin, body.Destination, body.Options)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}

	json.NewEncoder(w).Encode(plan)
}
