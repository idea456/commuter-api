package main

import (
	"context"
	"log"

	"github.com/idea456/commuter-api/pkg/seeder"
	"github.com/idea456/commuter-api/pkg/services"
	"github.com/idea456/commuter-api/pkg/transport"
)

func main() {
	ctx := context.Background()
	client, err := transport.NewNeo4JClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	directionsSvc := services.NewDirectionService()
	seeder := seeder.NewSeeder(client, directionsSvc)

	seeder.DropDatabase(ctx)
	seeder.SeedStops(ctx)
	seeder.SeedTrips(ctx)
	seeder.SeedProperties(ctx)
	seeder.SeedNearbyStops(ctx, 500)
}
