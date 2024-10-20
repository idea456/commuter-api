package transport

import (
	"context"
	"fmt"
	"slices"

	"log/slog"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

type Neo4JClient struct {
	Client neo4j.DriverWithContext
}

func NewNeo4JClient(ctx context.Context) (*Neo4JClient, error) {
	dbUri := "neo4j://localhost:7687"
	dbUser := "neo4j"
	dbPassword := "Abcd1234"

	driver, err := neo4j.NewDriverWithContext(
		dbUri,
		neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		return nil, err
	}

	err = driver.VerifyConnectivity(ctx)
	if err != nil {
		return nil, err
	}
	slog.Info("Connection established to Neo4J.")

	return &Neo4JClient{
		Client: driver,
	}, nil
}

func (c *Neo4JClient) Disconnect(ctx context.Context) {
	c.Client.Close(ctx)
}

func (c *Neo4JClient) GetNearestStops(ctx context.Context, coor models.Coordinate, distance int) ([]models.Stop, error) {
	query := "MATCH (s:Stop) WHERE point.distance(s.location, point({latitude: $latitude, longitude: $longitude})) < $distance RETURN s;"

	results, err := neo4j.ExecuteQuery(ctx, c.Client, query, map[string]any{
		"latitude":  coor.Latitude,
		"longitude": coor.Longitude,
		"distance":  distance,
	}, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	stops := make([]models.Stop, 0)
	for _, record := range results.Records {
		row, _ := record.Get("s")
		node := row.(dbtype.Node)
		stop := node.GetProperties()
		// fmt.Println(node.GetProperties()["name"])
		stops = append(stops, models.Stop{
			Name:      stop["name"].(string),
			Latitude:  stop["latitude"].(float64),
			Longitude: stop["longitude"].(float64),
		})
	}

	return stops, nil
}

func (c *Neo4JClient) GetBestPropertiesToStop(ctx context.Context, stop models.Stop, depth int) ([]models.Property, error) {
	TRANSIT_DEPTH := 4
	query := fmt.Sprintf("MATCH (p:Stop {name: $name})-[route*1..$f]-(nearby) RETURN p, nearby;", TRANSIT_DEPTH)

	results, err := neo4j.ExecuteQuery(ctx, c.Client, query, map[string]any{
		"name": stop.Name,
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
			properties = append(properties, models.Property{
				Name: property.(string),
			})
		}
	}

	return properties, nil
}
