package services

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"maps"
	"slices"
	"strings"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/idea456/commuter-api/pkg/transport"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
	"go.mongodb.org/mongo-driver/bson"
)

type PropertyService struct {
	MongoDBClient *transport.MongoDBClient
	Neo4JClient   *transport.Neo4JClient
}

type NearbyEdge struct {
	WalkDistance float64 `json:"walk_distance"`
	WalkTime     float64 `json:"walk_time"`
}

type WalkableStop struct {
	Stop         models.Stop
	WalkDistance float64
}

type TransferRange struct {
	MinTransfer int `json:"min_transfer"`
	MaxTranfer  int `json:"max_transfer"`
}

type FindTransitablePropertiesOptions struct {
	Range        TransferRange     `json:"transfer_range"`
	Origin       models.Coordinate `json:"origin"`
	WalkDistance int               `json:"walk_distance"`
}

type FindTransitablePropertiesByStop struct {
	WalkableStop WalkableStop  `json:"stop"`
	Range        TransferRange `json:"transfer_range"`
}

type FindNearestPropertiesResult struct {
	FastestByWalk  []models.Property
	FastestByTrain []models.Property
	FastestAverage []models.Property
}

// For pricing indexing:
// Property X
// 0 - 1,000: Listing[]
// 1,000 - 1,500: Listing[]
// 1,500 - 2,000: Listing[]
// 2,000 - 2,500: Listing[]
// 2,500 - 3,000: Listing[]

// Destination Y (split these into separate calls)
// 1km - walkTolerantProperties: Property[]
// 1km - 2km: busTolerantProperties[]
// 2km and beyond - trainTolerantProperties: Property[] (Query nearest train station properties)

// Keep a list of properties very near train stations
// actually, should be nearest train stations ALONG routes connecting to the station
// Basically breadth first search with a minimum depth and distance! Like Dijkstra
// Bukit bintang station: Property[5], like nearest train station here would be Tun Razak Exchange, Raja Chulan, Merdeka, then we find nearest properties around those stations!
// Cyberjaya City Center station: Property[5]
// So, 1. Find the nearest stations near the destination. This is for walking TO the destination from these stations
// 2. Using the transit graph, find 5 nearest connecting train stations
// 3. Find nearest properties near those connecting train stations
// 4. GGEZ!

// Keep a list of properties very near bus stops

// 1. Use Redis, query the top 3 nearest train stations at the destination
// 2. For each train station, get the top 3 nearest properties
// 3. For each property, get the itineary with highest train tolerance
// Total we'll have 9 properties for 2km and beyond

// Algorithm
// 1km radius (from nearest to furthest), for each property
// 1. Does it match the price range?
//	 - If yes, sort properties by high walk tolerance first, by the amount of time it takes to walk to the destination, walkableProperties
//	 - If no, search other properties
// If there are no good properties in 1km range, search in 1-2km range
//	- From here, walk tolerance is low, and subway/bus tolerance is high

func NewPropertyService() *PropertyService {
	graphClient, err := transport.NewNeo4JClient(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	return &PropertyService{
		Neo4JClient: graphClient,
	}
}

func (svc *PropertyService) GetProperty(id string) models.Property {
	db, err := transport.NewMongoDBClient()
	if err != nil {
		log.Fatal(err)
	}

	result := db.FindOne("commuter-db", "properties", bson.M{"id": id})

	var property models.Property
	err = result.Decode(&property)
	if err != nil {
		log.Fatal("wtf", err)
	}

	return property
}

func (svc *PropertyService) GetPropertiesByCell(origin models.Coordinate, cellId string) []models.Property {
	cursor, err := svc.MongoDBClient.Find("commuter-db", "properties", bson.M{"cellId": cellId})
	if err != nil {
		log.Fatal(err)
	}

	var results []models.Property
	cursor.All(context.TODO(), &results)

	return results
}

func (svc *PropertyService) FindNearestProperties(origin models.Coordinate, filter models.FindNearestPropertiesFilter) ([]models.Property, error) {
	redisClient, err := transport.NewRedisClient()
	if err != nil {
		return nil, err
	}
	var searchRadius float64 = 1
	if filter.Radius != 0 {
		searchRadius = filter.Radius
	}

	var nearestProperties []models.Property
	nearestPropertiesLocation := redisClient.RadiusSearch(origin, searchRadius)
	for _, nearestLocation := range nearestPropertiesLocation {
		tokens := strings.Split(nearestLocation.Name, ":")
		propertyId := tokens[1]
		property := svc.GetProperty(propertyId)

		if (filter.MinPrice == 0 || property.RentalRange.FromPrice >= filter.MinPrice) && (filter.MaxPrice == 0 || property.RentalRange.FromPrice <= filter.MaxPrice) {
			property.Distance = float64(nearestLocation.Dist)
			nearestProperties = append(nearestProperties, property)
		}
	}

	return nearestProperties, nil
}

func nodeToStop(node dbtype.Node) models.Stop {
	parameters := node.GetProperties()
	elementId := node.GetElementId()
	stopName := parameters["name"].(string)

	// stop ID can be an array due to stop nodes being merged
	var stopId []string
	if castedStopId, ok := parameters["stop_id"].([]string); ok {
		stopId = castedStopId
	} else {
		if castedStopId, ok := parameters["stop_id"].(string); ok {
			stopId = []string{castedStopId}
		}
	}

	var displayName string
	if castedDisplayName, ok := parameters["display_name"].([]string); ok {
		displayName = castedDisplayName[0]
	} else {
		displayName = parameters["display_name"].(string)
	}
	prefixDuration := parameters["prefix_duration"].([]interface{})
	prefixDurationCasted := make([]int, 0)
	for _, duration := range prefixDuration {
		durationCasted := int(duration.(int64))
		prefixDurationCasted = append(prefixDurationCasted, durationCasted)
	}
	suffixDuration := parameters["suffix_duration"].([]interface{})
	suffixDurationCasted := make([]int, 0)
	for _, duration := range suffixDuration {
		durationCasted := int(duration.(int64))
		suffixDurationCasted = append(suffixDurationCasted, durationCasted)
	}
	latitude := parameters["latitude"].(float64)
	longitude := parameters["longitude"].(float64)
	stop := models.Stop{
		ElementId:      elementId,
		Id:             stopId,
		Name:           stopName,
		DisplayName:    displayName,
		PrefixDuration: prefixDurationCasted,
		SuffixDuration: suffixDurationCasted,
		Coordinate: models.Coordinate{
			Latitude:  latitude,
			Longitude: longitude,
		},
	}
	return stop
}

func nodeToProperty(node dbtype.Node) models.Property {
	parameters := node.GetProperties()
	propertyName := parameters["name"]
	coordinates := parameters["coordinates"].([]any)

	latitude := coordinates[0].(float64)
	longitude := coordinates[1].(float64)

	property := models.Property{
		Id:       parameters["property_id"].(string),
		Name:     propertyName.(string),
		Region:   parameters["region"].(string),
		District: parameters["district"].(string),
		Address:  parameters["address"].(string),
		Type:     parameters["type"].(string),
		Coordinates: models.Coordinate{
			Latitude:  latitude,
			Longitude: longitude,
		},
	}

	return property
}

func (svc *PropertyService) FindTransitableProperties(ctx context.Context, options FindTransitablePropertiesOptions) ([]models.TransitableProperty, error) {
	nearestStops, err := svc.FindWalkableStationsByOrigin(ctx, options.Origin, options.WalkDistance)
	if err != nil {
		return nil, err
	}

	properties := make([]models.TransitableProperty, 0)
	checkedProperties := make(map[string]string)
	for _, nearestStop := range nearestStops {
		transitableProperties, err := svc.FindTransitablePropertiesByStop(ctx, FindTransitablePropertiesByStop{
			WalkableStop: nearestStop,
			Range:        options.Range,
		})

		if err != nil {
			return nil, err
		}

		// TODO: Can do checks like price here, or update Query filter
		for _, transitableProperty := range transitableProperties {
			if _, exists := checkedProperties[transitableProperty.Property.Name]; !exists {
				properties = append(properties, transitableProperty)
				checkedProperties[transitableProperty.Property.Name] = transitableProperty.Property.Name
			}
		}
	}

	return properties, nil
}

func calculateScore(originStop models.Stop, commutingStop models.Stop, walkDistanceToOriginStop float64, walkDistanceToCommutingStop float64) float64 {
	var commuteDuration int
	if originStop.PrefixDuration[0] > commutingStop.PrefixDuration[0] {
		commuteDuration = originStop.PrefixDuration[0] - commutingStop.PrefixDuration[0]
	} else {
		commuteDuration = originStop.SuffixDuration[0] - commutingStop.SuffixDuration[0]
	}

	return walkDistanceToOriginStop + walkDistanceToCommutingStop + float64(commuteDuration)
}

func (svc *PropertyService) FindTransitablePropertiesByStop(ctx context.Context, options FindTransitablePropertiesByStop) ([]models.TransitableProperty, error) {
	query := fmt.Sprintf("MATCH (p:Stop {name: $name})<-[routes*%d..%d]-(nearby) RETURN p, nearby, routes;", options.Range.MinTransfer, options.Range.MaxTranfer)

	results, err := neo4j.ExecuteQuery(ctx, svc.Neo4JClient.Client, query, map[string]any{
		"name": options.WalkableStop.Stop.Name,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	stopsMap := make(map[string]models.Stop)
	propertiesMap := make(map[string]models.Property)
	transitablesPropertiesMap := make(map[string]models.TransitableProperty)

	for _, record := range results.Records {
		row, _ := record.Get("nearby")
		node := row.(dbtype.Node)
		labels := node.Labels
		elementId := node.GetElementId()

		if slices.Contains(labels, "Property") {
			if _, exists := propertiesMap[elementId]; !exists {
				transitableProperty := nodeToProperty(node)
				propertiesMap[elementId] = transitableProperty
			}
		} else if slices.Contains(labels, "Stop") {
			if _, exists := stopsMap[elementId]; !exists {
				stop := nodeToStop(node)
				stopsMap[elementId] = stop
			}
		}
	}

	// Calculate scores for each property
	for _, record := range results.Records {
		routes, _ := record.Get("routes")
		for _, route := range routes.([]interface{}) {
			rel := route.(dbtype.Relationship)
			if rel.Type == "NEARBY" {
				if nearestStopToProperty, exists := stopsMap[rel.EndElementId]; exists {
					propertyNode, propertyExists := propertiesMap[rel.StartElementId]
					if !propertyExists {
						slog.Info(fmt.Sprintf("not found %s\n", rel.StartElementId))
						continue
					}

					parameters := rel.GetProperties()
					walkTimeToCommutingStop := parameters["walk_time"].(float64)
					walkDistanceToCommutingStop := parameters["walk_distance"].(float64)
					score := calculateScore(options.WalkableStop.Stop, nearestStopToProperty, options.WalkableStop.WalkDistance, walkTimeToCommutingStop)

					transitablesPropertiesMap[rel.StartElementId] = models.TransitableProperty{
						Property:                  propertyNode,
						Score:                     score,
						NearestStop:               nearestStopToProperty,
						WalkDistanceToNearestStop: walkDistanceToCommutingStop,
						WalkTimeToCommutingStop:   walkTimeToCommutingStop,
					}
				} else {
					// slog.Info("stop not found")
				}
			}

		}
	}

	properties := maps.Values(transitablesPropertiesMap)
	sortedByScore := slices.SortedFunc(properties, func(prop1, prop2 models.TransitableProperty) int {
		return int(prop1.Score) - int(prop2.Score)
	})

	return sortedByScore, nil
}

func (svc *PropertyService) FindWalkablePropertiesByOrigin(ctx context.Context, origin models.Coordinate, maxWalkDistance int) ([]models.Property, error) {
	query := "MATCH (p:Property) WHERE point.distance(p.location, point({latitude:$latitude, longitude:$longitude})) < $maxWalkableDistance RETURN p"

	results, err := neo4j.ExecuteQuery(ctx, svc.Neo4JClient.Client, query, map[string]any{
		"latitude":            origin.Latitude,
		"longitude":           origin.Longitude,
		"maxWalkableDistance": maxWalkDistance,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	properties := make([]models.Property, 0)
	for _, record := range results.Records {
		node := record.AsMap()["p"].(dbtype.Node)
		labels := node.Labels

		if slices.Contains(labels, "Property") {
			property := nodeToProperty(node)
			properties = append(properties, property)
		}
	}
	return properties, nil
}

func (svc *PropertyService) FindWalkableStationsByOrigin(ctx context.Context, origin models.Coordinate, maxWalkDistance int) ([]WalkableStop, error) {
	query := "MATCH (p:Stop) WITH point.distance(p.location, point({latitude:$latitude, longitude:$longitude})) as dist, p WHERE dist < $maxWalkableDistance ORDER BY dist RETURN p, dist"

	results, err := neo4j.ExecuteQuery(ctx, svc.Neo4JClient.Client, query, map[string]any{
		"latitude":            origin.Latitude,
		"longitude":           origin.Longitude,
		"maxWalkableDistance": maxWalkDistance,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	stops := make([]WalkableStop, 0)
	for _, record := range results.Records {
		node := record.AsMap()["p"].(dbtype.Node)
		labels := node.Labels
		walkDistance, _ := record.Get("dist")

		if slices.Contains(labels, "Stop") {
			stop := nodeToStop(node)
			stops = append(stops, WalkableStop{
				Stop:         stop,
				WalkDistance: walkDistance.(float64),
			})
		}
	}
	return stops, nil
}
