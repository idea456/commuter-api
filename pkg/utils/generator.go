package utils

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/idea456/commuter-api/pkg/transport"
)

type SeedGenerator struct{}

func NewSeedGenerator() *SeedGenerator {
	return &SeedGenerator{}
}

func (g *SeedGenerator) GenerateStops() error {
	file, err := os.Open("pkg/static/gtfs_rapid_rail_kl/stops.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	r := csv.NewReader(file)
	data, err := r.ReadAll()
	if err != nil {
		return err
	}

	fmt.Println("CREATE")
	existsMap := make(map[string]bool)

	for i, row := range data {
		// for _, col := range row {
		// 	fmt.Printf(",", col)
		// }
		if i > 0 {
			stopName := strings.Replace(strings.Trim(row[9], " "), " ", "_", -1)
			stopName = strings.Replace(stopName, "'", "", -1)
			stopName = strings.Replace(stopName, "-", "_", -1)
			_, exists := existsMap[stopName]
			if !exists {
				fmt.Printf("(%s:Stop {name: '%s', location: point({latitude: %s, longitude: %s})}),\n", stopName, stopName, row[2], row[3])
				existsMap[stopName] = true
			}
			// stopName = fmt.Sprintf("%s_%s", stopName, row[0])
			// fmt.Printf("CREATE (%s: Stop {id: '%s', name: '%s', longitude: %s, latitude: %s })\n", row[0], row[0], stopName, row[3], row[2])
			// fmt.Printf("CREATE (%s:Stop)\nSET %s.name ='%s',\n%s.location = point({latitude: %s, longitude: %s})\nRETURN %s\n", stopName, stopName, stopName, stopName, row[2], row[3], stopName)
		}
	}

	return nil
}

func getStopNameMapping() (map[string]string, error) {
	file, err := os.Open("pkg/static/gtfs_rapid_rail_kl/stops.txt")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	data, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	mapping := make(map[string]string)

	for i, row := range data {
		if i > 0 {
			stopName := strings.Replace(strings.Trim(row[9], " "), " ", "_", -1)
			stopName = strings.Replace(stopName, "'", "", -1)
			stopName = strings.Replace(stopName, "-", "_", -1)
			// stopName = fmt.Sprintf("%s_%s", stopName, row[0])

			mapping[row[0]] = stopName
		}
	}
	return mapping, nil
}

func filterWeekendTrips(stopTimes [][]string) (filteredStopTimes [][]string) {
	for _, stopTime := range stopTimes {
		tripId := stopTime[2]
		if strings.Contains(tripId, "MonFri") {
			filteredStopTimes = append(filteredStopTimes, stopTime)
		}
	}
	return filteredStopTimes
}

func (g *SeedGenerator) GenerateEdges() error {
	mapping, _ := getStopNameMapping()

	file, _ := os.Open("pkg/static/gtfs_rapid_rail_kl/stop_times.txt")
	defer file.Close()

	r := csv.NewReader(file)
	data, _ := r.ReadAll()

	data = filterWeekendTrips(data)

	var currentRoute string
	// fmt.Println("CREATE")
	for i, row := range data {
		if currentRoute == "" {
			currentRoute = row[0]
		} else if currentRoute != row[0] {
			currentRoute = row[0]
		}

		// currentTrip := row[2]
		var nextStop string
		if i < len(data)-1 && data[i+1][0] == currentRoute {
			nextStop = mapping[data[i+1][5]]
		}

		currentStop := mapping[row[5]]
		directionId := row[1]
		if i > 0 && nextStop != "" && currentStop != nextStop {
			duration, _ := strconv.Atoi(row[8])
			if directionId == "0" {
				fmt.Printf("(%s)-[:%s_ROUTE {id: '%s', duration: %d}]->(%s),\n", currentStop, currentRoute, currentRoute, duration, nextStop)
			} else {
				fmt.Printf("(%s)-[:%s_ROUTE {id: '%s', duration: %d}]->(%s),\n", nextStop, currentRoute, currentRoute, duration, currentStop)
			}
		}
	}

	return nil
}

func (g *SeedGenerator) GenerateProperties() error {
	properties, _ := LoadPropertiesJSON()

	fmt.Println("CREATE")
	for _, property := range properties {
		fmt.Printf("(:Property {id: \"%s\", name: \"%s\", address: \"%s\", link: \"%s\", district: \"%s\", priceRange: [0, 0, 0, 0, 0], coordinates: [%f, %f], location: point({latitude: %f, longitude: %f})}),\n", property.Id, property.Name, property.Address, property.Link, property.District, property.Coordinates.Latitude, property.Coordinates.Longitude, property.Coordinates.Latitude, property.Coordinates.Longitude)
	}

	return nil
}

func (g *SeedGenerator) GenerateNearestStops(distance int) error {
	ctx := context.Background()
	client, err := transport.NewNeo4JClient(ctx)
	defer client.Disconnect(ctx)

	if err != nil {
		slog.Error(err.Error())
		return err
	}

	properties, _ := LoadPropertiesJSON()

	for _, property := range properties {
		nearestStops, _ := client.GetNearestStops(ctx, property.Coordinates, distance)
		for _, nearestStop := range nearestStops {
			fmt.Printf("MATCH (p:Property {name: \"%s\"}), (s:Stop {name:\"%s\"}) CREATE (p)-[:NEARBY]->(s);\n", property.Name, nearestStop.Name)
		}
	}

	return nil
}
