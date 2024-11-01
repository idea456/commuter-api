package seeder

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

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
		// Skip header
		if i > 0 {
			stopName := strings.Replace(strings.Trim(row[9], " "), " ", "_", -1)
			stopName = strings.Replace(stopName, "'", "", -1)
			stopName = strings.Replace(stopName, "-", "_", -1)

			query := "CREATE (:Stop {stop_id: $stop_id, name: $name, display_name: $display_name, prefix_duration: $prefix_duration, suffix_duration: $suffix_duration, latitude: $latitude, longitude: $longitude, location: point({latitude: $latitude, longitude: $longitude})});"

			latitude, _ := strconv.ParseFloat(row[2], 64)
			longitude, _ := strconv.ParseFloat(row[3], 64)

			_, err := neo4j.ExecuteQuery(ctx, g.client.Client, query, map[string]any{
				"stop_id":         row[0],
				"display_name":    row[9],
				"name":            row[1],
				"latitude":        latitude,
				"longitude":       longitude,
				"prefix_duration": "",
				"suffix_duration": "",
			}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
			if err != nil {
				slog.Error(fmt.Sprintf("unable to seed %s: %v", stopName, err))
				continue
			}
			existsMap[stopName] = true

		}
	}

	return nil
}

func (g *Seeder) MergeSimilarStops(ctx context.Context) {
	file, err := os.Open("pkg/static/gtfs_rapid_rail_kl/stops.txt")
	if err != nil {
		slog.Error(err.Error())
		return
	}
	defer file.Close()

	r := csv.NewReader(file)
	data, err := r.ReadAll()
	if err != nil {
		slog.Error(err.Error())
		return
	}

	mapping := make(map[string][]models.Stop)

	for i, row := range data {
		if i > 0 {
			name := row[1]
			stop := models.Stop{
				Id:          []string{row[0]},
				DisplayName: row[9],
				Name:        row[1],
			}
			if _, exists := mapping[name]; exists {
				mapping[name] = append(mapping[name], stop)
			} else {
				mapping[name] = []models.Stop{stop}
			}
		}
	}

	for _, stops := range mapping {
		if len(stops) > 1 {
			query := "MATCH "
			variables := ""
			parameters := make(map[string]any)
			for i, stop := range stops {
				parameters[fmt.Sprintf("stop_id_%d", i)] = stop.Id
				if i < len(stops)-1 {
					query += fmt.Sprintf("(p%d:Stop {stop_id: $stop_id_%d}), ", i, i)
					variables += fmt.Sprintf("p%d,", i)
				} else {
					query += fmt.Sprintf("(p%d:Stop {stop_id: $stop_id_%d})", i, i)
					variables += fmt.Sprintf("p%d", i)
				}
			}
			variables = fmt.Sprintf("[%s]", variables)
			query = fmt.Sprintf("%s WITH head(collect(%s)) as nodes CALL apoc.refactor.mergeNodes(nodes,{properties:{ name: 'discard', latitude: 'discard', longitude: 'discard', location: 'discard', walk_time: 'discard', walk_distance: 'discard', `.*`: \"combine\" }, mergeRels:true}) YIELD node RETURN node;", query, variables)

			_, err = neo4j.ExecuteQuery(ctx, g.client.Client, query, parameters, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
			if err != nil {
				slog.Error(fmt.Sprintf("unable to merge stops: %v", err))
				continue
			}
		}
	}
}

func computeTransfers() {}

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

func strToTime(timeString string) (time.Time, error) {
	layout := "15:04:05"
	t, err := time.Parse(layout, timeString)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// route_id,direction_id,trip_id,arrival_time,departure_time,stop_id,stop_sequence
func (g *Seeder) SeedTrips(ctx context.Context) error {
	file, _ := os.Open("pkg/static/gtfs_rapid_rail_kl/stop_times.txt")
	defer file.Close()

	r := csv.NewReader(file)
	data, _ := r.ReadAll()

	data = filterWeekendTrips(data)

	// Prefix/Suffix duration is used to calculate how much duration in seconds  it will take to reach to a particular stop from the very first stop
	// This is gonna be used fo
	var prefixDuration int
	var suffixDuration int
	var duration int
	var previousArrivalTime time.Time
	var currentRoute string
	for i, row := range data {
		if currentRoute == "" || currentRoute != row[0] {
			currentRoute = row[0]
		}

		stopSequence, err := strconv.Atoi(data[i][6])
		if err != nil {
			log.Fatal(fmt.Errorf("could not convert %s to integer: %w", data[i][6], err))
		}
		directionId, err := strconv.Atoi(data[i][1])
		if err != nil {
			log.Fatal(fmt.Errorf("could not convert directionId %s to integer: %w", data[i][6], err))
		}

		arrivalTime, err := strToTime(data[i][3])
		if err != nil {
			log.Fatal(fmt.Errorf("could not parse %s to time: %v", data[i][3], err))
		}

		if stopSequence == 1 {
			if directionId == 0 {
				prefixDuration = 0 // First stop doesn't have prefix duration, since its literally the first stop
			} else {
				suffixDuration = 0
			}
		} else {
			duration = int(arrivalTime.Sub(previousArrivalTime).Seconds())
			if directionId == 0 {
				prefixDuration += duration
			} else {
				suffixDuration += duration
			}
		}
		previousArrivalTime = arrivalTime

		var nextStop string
		if i < len(data)-1 && data[i+1][0] == currentRoute {
			nextStop = data[i+1][5]
		}

		currentStop := row[5]
		var query string
		var parameters map[string]any

		if directionId == 0 {
			query = "MATCH (s:Stop {stop_id:$start_stop}) SET s.prefix_duration=$prefix_duration;"
			parameters = map[string]any{
				"start_stop":      currentStop,
				"prefix_duration": []int{prefixDuration},
			}
		} else {
			query = "MATCH (s:Stop {stop_id:$start_stop}) SET s.suffix_duration=$suffix_duration;"
			parameters = map[string]any{
				"start_stop":      currentStop,
				"suffix_duration": []int{suffixDuration},
			}
		}
		_, err = neo4j.ExecuteQuery(ctx, g.client.Client, query, parameters, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
		if err != nil {
			slog.Error(fmt.Sprintf("unable to set prefix/suffix duration for stop %s: %v", currentStop, err))
			continue
		}

		if i > 0 && nextStop != "" && currentStop != nextStop {
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
			directions, _ := g.directionsSvc.GetDirections(property.Coordinates, models.Coordinate{
				Latitude:  nearestStop.Latitude,
				Longitude: nearestStop.Longitude,
			}, models.DirectionOptions{
				WalkReluctance: 0,
				TransportModes: []string{"WALK"},
			})

			if len(directions.Itineraries) == 0 {
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
