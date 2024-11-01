package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/idea456/commuter-api/pkg/controllers"
	"github.com/rs/cors"
)

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}



// func rateLimiter(next func(w http.ResponseWriter, r *http.Request) http.Handler {
// 	limiter := rate.NewLimiter(20, 4)
// })

func init() {}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	propertiesController := controllers.NewPropertiesController()
	mux.HandleFunc("/properties/nearest/walkable", propertiesController.GetWalkableProperties)
	mux.HandleFunc("/properties/nearest/transit", propertiesController.GetTransitableProperties)
	mux.HandleFunc("/properties/([^/]+)", propertiesController.GetProperty)

	directionsController, err := controllers.NewDirectionsController()
	if err != nil {
		log.Fatalf("there was an error in initialising the directions service: %v", err)
		return
	}
	mux.HandleFunc("/directions", directionsController.GetDirections)

	handler := cors.Default().Handler(mux)

	fmt.Println("Listening at port 4001...")
	err = http.ListenAndServe(":4001", handler)
	if err != nil {
		log.Fatal(err)
	}
}
