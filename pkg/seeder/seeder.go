package seeder

import (
	"context"
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/idea456/commuter-api/pkg/services"
	"github.com/idea456/commuter-api/pkg/transport"
	"github.com/idea456/commuter-api/pkg/utils"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Seeder struct {
	client        *transport.Neo4JClient
	directionsSvc *services.DirectionService
}

func NewSeeder(client *transport.Neo4JClient, directionsClient *services.DirectionService) *Seeder {
	return &Seeder{
		client:        client,
		directionsSvc: directionsClient,
	}
}

func (g *Seeder) DropDatabase(ctx context.Context) error {
	_, err := neo4j.ExecuteQuery(ctx, g.client.Client, "MATCH (n) DETACH DELETE n;", nil, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	return err
}

func (g *Seeder) SeedStops(ctx context.Context) error {
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

	existsMap := make(map[string]bool)

	for i, row := range data {
		if i > 0 {
			stopName := strings.Replace(strings.Trim(row[9], " "), " ", "_", -1)
			stopName = strings.Replace(stopName, "'", "", -1)
			stopName = strings.Replace(stopName, "-", "_", -1)
			_, exists := existsMap[stopName]
			if !exists {
				query := "CREATE (:Stop {stop_id: $stop_id, name: $name, latitude: $latitude, longitude: $longitude, location: point({latitude: $latitude, longitude: $longitude})});"

				latitude, _ := strconv.ParseFloat(row[2], 64)
				longitude, _ := strconv.ParseFloat(row[3], 64)

				_, err := neo4j.ExecuteQuery(ctx, g.client.Client, query, map[string]any{
					"stop_id":   row[0],
					"name":      row[9],
					"latitude":  latitude,
					"longitude": longitude,
				}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
				if err != nil {
					slog.Error(fmt.Sprintf("unable to seed %s: %v", stopName, err))
					continue
				}
				existsMap[stopName] = true
			}
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

func (g *Seeder) SeedTrips(ctx context.Context) error {
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
			nextStop = data[i+1][5]
		}

		currentStop := row[5]
		var query string
		if i > 0 && nextStop != "" && currentStop != nextStop {
			duration, _ := strconv.Atoi(row[8])
			var parameters map[string]any
			query = "MATCH (s:Stop {stop_id:$start_stop}), (t:Stop {stop_id:$end_stop}) CREATE (s)-[:ROUTE {route_id: $route_id, direction_id: $direction_id, trip_id: $trip_id, name: $route_name, duration: $duration}]->(t);"
			parameters = map[string]any{
				"start_stop":   currentStop,
				"route":        currentRoute,
				"route_id":     currentRoute,
				"direction_id": row[1],
				"trip_id":      row[2],
				"route_name":   currentRoute,
				"duration":     duration,
				"end_stop":     nextStop,
			}

			_, err := neo4j.ExecuteQuery(ctx, g.client.Client, query, parameters, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
			if err != nil {
				slog.Error(fmt.Sprintf("unable to seed trip %s: %v", currentStop, err))
				continue
			}
		}
	}

	return nil
}

func (g *Seeder) SeedProperties(ctx context.Context) error {
	properties, _ := utils.LoadPropertiesJSON()

	for _, property := range properties {
		query := "CREATE (:Property {property_id: $property_id, name: $name, type: $type, address: $address, link: $link, district: $district, region: $region, facilities: $facilities, coordinates: [$latitude, $longitude], location: point({latitude: $latitude, longitude: $longitude})});"

		parameters := map[string]any{
			"property_id": property.Id,
			"name":        property.Name,
			"type":        property.Type,
			"address":     property.Address,
			"link":        property.Link,
			"region":      property.Region,
			"district":    property.District,
			"facilities":  property.Facilities,
			"latitude":    property.Coordinates.Latitude,
			"longitude":   property.Coordinates.Longitude,
		}
		_, err := neo4j.ExecuteQuery(ctx, g.client.Client, query, parameters, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
		if err != nil {
			slog.Error(fmt.Sprintf("unable to seed property %s: %v", property.Name, err))
			return err
		}

	}

	return nil
}

func (g *Seeder) SeedNearbyStops(ctx context.Context, distance int) error {
	_, err := neo4j.ExecuteQuery(ctx, g.client.Client, "DROP INDEX property_name_idx IF EXISTS;", nil, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(fmt.Sprintf("unable to drop index for properties: %v", err))
		return err
	}
	_, err = neo4j.ExecuteQuery(ctx, g.client.Client, "DROP INDEX stop_name_idx IF EXISTS;", nil, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(fmt.Sprintf("unable to drop index for stops: %v", err))
		return err
	}
	_, err = neo4j.ExecuteQuery(ctx, g.client.Client, "CREATE INDEX property_name_idx FOR (p:Property) ON (p.name);", nil, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(fmt.Sprintf("unable to create index for properties: %v", err))
		return err
	}

	_, err = neo4j.ExecuteQuery(ctx, g.client.Client, "CREATE INDEX stop_name_idx FOR (s:Stop) ON (s.name);", nil, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(fmt.Sprintf("unable to create index for stops: %v", err))
		return err
	}

	properties, _ := utils.LoadPropertiesJSON()
	for _, property := range properties {
		nearestStops, _ := g.client.GetNearestStops(ctx, property.Coordinates, distance)
		for _, nearestStop := range nearestStops {
			directions := g.directionsSvc.GetDirections(property.Coordinates, models.Coordinate{
				Latitude:  nearestStop.Latitude,
				Longitude: nearestStop.Longitude,
			}, models.DirectionOptions{
				WalkReluctance: 0,
				TransportModes: []string{"WALK"},
			})

			if len(directions.Itineraries) == 0 {
				fmt.Println(nearestStop.Latitude, nearestStop.Longitude)
				slog.Info(fmt.Sprintf("No directions found between stop %s to property %s...", nearestStop.Name, property.Name))
				continue
			}
			if len(directions.Itineraries) > 0 && directions.Itineraries[0].WalkDistance > float64(distance) {
				continue
			}
			slog.Info(fmt.Sprintf("Matching nearest stop %s to property %s...", nearestStop.Name, property.Name))

			query := "MATCH (p:Property {name: $property_name}), (s:Stop {name:$stop_name}) CREATE (p)-[:NEARBY { walk_distance: $walk_distance, walk_time: $walk_time }]->(s);"
			parameters := map[string]any{
				"property_name": property.Name,
				"walk_distance": directions.Itineraries[0].WalkDistance,
				"walk_time":     directions.Itineraries[0].WalkTime,
				"stop_name":     nearestStop.Name,
			}
			_, err := neo4j.ExecuteQuery(ctx, g.client.Client, query, parameters, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
			if err != nil {
				slog.Error(fmt.Sprintf("unable to seed property %s: %v", property.Name, err))
				return err
			}
		}
	}

	return nil
}
