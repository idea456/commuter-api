package utils

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/idea456/commuter-api/pkg/models"
)

type ScrapedProperty struct {
	Id          string            `json:"id" bson:"id"`
	District    string            `json:"district" bson:"district"`
	Name        string            `json:"name" bson:"name"`
	Address     string            `json:"address" bson:"address"`
	Facilities  []string          `json:"facilities" bson:"facilities"`
	Link        string            `json:"link" bson:"link"`
	Type        string            `json:"type" bson:"type"`
	Coordinates models.Coordinate `json:"coordinates" bson:"coordinates"`
}

func LoadPropertiesJSON() ([]models.Property, error) {
	raw, err := ioutil.ReadFile("./properties.json")
	if err != nil {
		return nil, err
	}

	var scrapedProperties []ScrapedProperty
	json.Unmarshal(raw, &scrapedProperties)

	properties := make([]models.Property, 0)
	for _, scrapedProperty := range scrapedProperties {
		if scrapedProperty.Coordinates.Latitude == 0 && scrapedProperty.Coordinates.Longitude == 0 {
			continue
		}
		idTokens := strings.Split(scrapedProperty.Link, "/")

		properties = append(properties, models.Property{
			Id:          idTokens[len(idTokens)-1],
			District:    scrapedProperty.District,
			Address:     scrapedProperty.Address,
			Facilities:  scrapedProperty.Facilities,
			Coordinates: scrapedProperty.Coordinates,
			Link:        scrapedProperty.Link,
			Type:        scrapedProperty.Type,
			Name:        scrapedProperty.Name,
		})
	}

	return properties, nil
}
