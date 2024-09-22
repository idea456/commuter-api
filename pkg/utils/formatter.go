package utils

import (
	"encoding/json"
	"io/ioutil"

	"github.com/idea456/commuter-api/pkg/models"
)

func LoadPropertiesJSON() ([]models.Property, error) {
	raw, err := ioutil.ReadFile("./properties.json")
	if err != nil {
		return nil, err
	}

	var properties []models.Property
	json.Unmarshal(raw, &properties)

	return properties, nil
}
