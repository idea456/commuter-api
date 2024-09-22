package services

import (
	"context"
	"log"
	"log/slog"
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
	client, err := transport.NewMongoDBClient()
	if err != nil {
		log.Fatal(err)
	}

	graphClient, err := transport.NewNeo4JClient(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	return &PropertyService{
		MongoDBClient: client,
		Neo4JClient:   graphClient,
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

func (svc *PropertyService) FindTransitableProperties(ctx context.Context, origin models.Coordinate) ([]models.Property, error) {
	nearestStops, err := svc.FindWalkableStationsByOrigin(ctx, origin)
	if err != nil {
		return nil, err
	}

	properties := make([]models.Property, 0)
	checkedProperties := make(map[string]string)
	for _, nearestStop := range nearestStops {
		transitableProperties, err := svc.FindTransitablePropertiesByStation(ctx, origin, nearestStop)
		if err != nil {
			return nil, err
		}

		// TODO: Can do checks like price here, or update Query filter
		for _, transitableProperty := range transitableProperties {
			if _, exists := checkedProperties[transitableProperty.Name]; !exists {
				properties = append(properties, transitableProperty)
				checkedProperties[transitableProperty.Name] = transitableProperty.Name
			}
		}
	}

	return properties, nil
}

func (svc *PropertyService) FindTransitablePropertiesByStation(ctx context.Context, origin models.Coordinate, station models.Stop) ([]models.Property, error) {
	// SEARCH_DEPTH := 2
	query := "MATCH (p:Stop {name: $name})-[route*1..2]-(nearby) RETURN p, nearby;"

	results, err := neo4j.ExecuteQuery(ctx, svc.Neo4JClient.Client, query, map[string]any{
		"name": station.Name,
		// "depth": SEARCH_DEPTH,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	properties := make([]models.Property, 0)
	for _, record := range results.Records {
		row, _ := record.Get("nearby")
		node := row.(dbtype.Node)
		labels := node.Labels
		if slices.Contains(labels, "Property") {
			property := node.GetProperties()["name"]
			coordinates := node.GetProperties()["coordinates"].([]any)

			latitude := coordinates[0].(float64)
			longitude := coordinates[1].(float64)

			properties = append(properties, models.Property{
				Id:       node.GetProperties()["id"].(string),
				Name:     property.(string),
				District: node.GetProperties()["district"].(string),
				Address:  node.GetProperties()["address"].(string),
				Coordinates: models.Coordinate{
					Latitude:  latitude,
					Longitude: longitude,
				},
			})
		}
	}
	return properties, nil
}

func (svc *PropertyService) FindWalkablePropertiesByOrigin(ctx context.Context, origin models.Coordinate) ([]models.Property, error) {
	MAX_WALKABLE_DISTANCE := 1000
	query := "MATCH (p:Property) WHERE point.distance(p.location, point({latitude:$latitude, longitude:$longitude})) < $maxWalkableDistance RETURN p"

	results, err := neo4j.ExecuteQuery(ctx, svc.Neo4JClient.Client, query, map[string]any{
		"latitude":            origin.Latitude,
		"longitude":           origin.Longitude,
		"maxWalkableDistance": MAX_WALKABLE_DISTANCE,
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
			property := node.GetProperties()["name"]
			properties = append(properties, models.Property{
				Id:       node.GetProperties()["id"].(string),
				Name:     property.(string),
				District: node.GetProperties()["district"].(string),
				Address:  node.GetProperties()["address"].(string),
				Coordinates: models.Coordinate{
					Latitude:  node.GetProperties()["coordinates"].([]float64)[0],
					Longitude: node.GetProperties()["coordinates"].([]float64)[1],
				},
			})
		}
	}
	return properties, nil
}

func (svc *PropertyService) FindWalkableStationsByOrigin(ctx context.Context, origin models.Coordinate) ([]models.Stop, error) {
	MAX_WALKABLE_DISTANCE := 1000
	query := "MATCH (p:Stop) WHERE point.distance(p.location, point({latitude:$latitude, longitude:$longitude})) < $maxWalkableDistance RETURN p"

	results, err := neo4j.ExecuteQuery(ctx, svc.Neo4JClient.Client, query, map[string]any{
		"latitude":            origin.Latitude,
		"longitude":           origin.Longitude,
		"maxWalkableDistance": MAX_WALKABLE_DISTANCE,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	stops := make([]models.Stop, 0)
	for _, record := range results.Records {
		node := record.AsMap()["p"].(dbtype.Node)
		labels := node.Labels

		if slices.Contains(labels, "Stop") {
			stop := node.GetProperties()["name"]
			stops = append(stops, models.Stop{
				Name: stop.(string),
			})
		}
	}
	return stops, nil
}
