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

func NewDirectionsController() *DirectionsController {
	return &DirectionsController{
		directionsService: services.NewDirectionService(),
	}
}

func (ctl *DirectionsController) GetDirections(w http.ResponseWriter, r *http.Request) {
	var body GetDirectionsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid body format", http.StatusBadRequest)
		return
	}

	plan := ctl.directionsService.GetDirections(body.Origin, body.Destination, body.Options)

	json.NewEncoder(w).Encode(plan)
}
