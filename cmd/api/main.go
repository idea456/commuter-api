package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/idea456/commuter-api/pkg/controllers"
	"github.com/idea456/commuter-api/pkg/transport"
	"github.com/idea456/commuter-api/pkg/utils"
	"github.com/rs/cors"
	"github.com/uber/h3-go/v4"
)

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func seed() {
	properties, err := utils.LoadPropertiesJSON()
	db, err := transport.NewMongoDBClient()

	if err != nil {
		log.Fatal(err)
	}

	db.DropDatabase("commuter-db")
	for _, property := range properties {
		log.Printf("Inserting %s...\n", property.Name)
		property.CellId = utils.ToCell(property.Coordinates).String()
		err = db.InsertItem("commuter-db", "properties", property)
		if err != nil {
			log.Fatal(err)
		}
	}

	client, err := transport.NewRedisClient()
	if err != nil {
		log.Fatal(err)
	}
	client.FlushAll()
	for _, property := range properties {
		log.Printf("Caching %s...\n", property.Name)
		cell := h3.LatLngToCell(h3.LatLng{
			Lat: property.Coordinates.Latitude,
			Lng: property.Coordinates.Longitude,
		}, 8)

		cellKey := fmt.Sprintf("cells:%s", cell.String())
		client.ListPush(cellKey, property.Id)

		client.GeoAdd(fmt.Sprintf("properties:%s:%s", property.Id, cell.String()), property.Coordinates)

	}
}

func main() {
	// seed()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	propertiesController := controllers.NewPropertiesController()
	mux.HandleFunc("/properties/nearest/walkable", propertiesController.GetWalkableProperties)
	mux.HandleFunc("/properties/nearest/transit", propertiesController.GetTransitableProperties)
	mux.HandleFunc("/properties/([^/]+)", propertiesController.GetProperty)

	directionsController := controllers.NewDirectionsController()
	mux.HandleFunc("/directions", directionsController.GetDirections)

	handler := cors.Default().Handler(mux)

	fmt.Println("Listening at port 4001...")
	err := http.ListenAndServe(":4001", handler)
	if err != nil {
		log.Fatal(err)
	}
}
