package services

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/idea456/commuter-api/pkg/transport"
)

type DirectionService struct {
	graphqlClient *transport.GraphQLClient
}

func NewDirectionService() *DirectionService {
	graphqlClient, err := transport.NewGraphQLClient()
	if err != nil {
		log.Fatal(err)
	}

	return &DirectionService{
		graphqlClient: graphqlClient,
	}
}

func (svc *DirectionService) GetDirections(origin models.Coordinate, destination models.Coordinate, options models.DirectionOptions) models.PlanDataResponse {
	var transportModes string
	if len(options.TransportModes) > 0 {
		transportModesFmt := make([]string, 0)
		for _, transportMode := range options.TransportModes {
			transportModesFmt = append(transportModesFmt, fmt.Sprintf("{mode: %s}", transportMode))
		}
		transportModes = fmt.Sprintf("[%s]", strings.Join(transportModesFmt, ","))
	} else {
		transportModes = "[\"WALK\", \"TRANSIT\"]"
	}

	query := fmt.Sprintf(`
		{
		plan(
			from: {lat: %f, lon: %f}
			to: {lat: %f, lon: %f}
			walkReluctance: %d
			transportModes: %s
		) {
			itineraries {
				start
				end
				duration
				walkTime
				walkDistance
				waitingTime
				legs {
					mode
					transitLeg
					from {
						name
					}
					to {
						name
					}
					duration
					legGeometry {
						length
						points
					}
					route {
						longName
						shortName
						color
					}
					distance
				}
			}
		}
	}
	`, origin.Latitude, origin.Longitude, destination.Latitude, destination.Longitude, options.WalkReluctance, transportModes)

	fmt.Println(query)
	response, err := svc.graphqlClient.Query(query)
	if err != nil {
		log.Fatal("cant get directions", err)
	}

	rawResponse, _ := json.Marshal(response)
	var planResponse models.PlanResponse
	json.Unmarshal(rawResponse, &planResponse)

	itinearies := planResponse.Plan.Itineraries

	if options.WalkReluctance >= 4 {
		sort.Slice(itinearies, func(i, j int) bool {
			return itinearies[i].WalkTime < itinearies[j].WalkTime
		})

		// // Remove routes with only walking
		// itineariesWithoutWalking := make([]models.Itineary, 0)
		// for _, itineary := range planResponse.Plan.Itineraries {
		// 	hasWalking := false
		// 	for _, leg := range itineary.Legs {
		// 		if leg.Mode == "WALK" {
		// 			hasWalking = true
		// 		}
		// 	}

		// 	if !hasWalking {
		// 		itineariesWithoutWalking = append(itineariesWithoutWalking, itineary)
		// 	}
		// }

		// itinearies = itineariesWithoutWalking
	}

	return models.PlanDataResponse{
		Itineraries: itinearies,
	}
}
