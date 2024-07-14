package services

import (
	"github.com/idea456/commuter-api/pkg/models"
)

type PropertyService struct {
}

func NewPropertyService() *PropertyService {
	return &PropertyService{}
}

func (svc *PropertyService) GetProperties() []models.Property {
	properties := make([]models.Property, 0)
	return properties
}

func (svc *PropertyService) GetProperty() models.Property {
	property := models.Property{}

	return property
}
