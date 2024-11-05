package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/idea456/commuter-api/pkg/transport"
)

type DirectionService struct {
	graphqlClient *transport.GraphQLClient
}

func NewDirectionService() (*DirectionService, error) {
	url := os.Getenv("OTP_GRAPHQL_URL")
	if url == "" {
		return nil, errors.New("GraphQL URL not specified in environment variable")
	}
	graphqlClient, err := transport.NewGraphQLClient(url)
	if err != nil {
		log.Fatal(err)
	}

	return &DirectionService{
		graphqlClient: graphqlClient,
	}, nil
}

func (svc *DirectionService) GetDirections(origin models.Coordinate, destination models.Coordinate, options models.DirectionOptions) (*models.PlanDataResponse, error) {
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
			transportModes: %s
			optimize: QUICK
		) {
			itineraries {
				start
				end
				duration
				walkTime
				walkDistance
				waitingTime
				numberOfTransfers
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
	`, origin.Latitude, origin.Longitude, destination.Latitude, destination.Longitude, transportModes)

	response, err := svc.graphqlClient.Query(query)
	if err != nil {
		return nil, fmt.Errorf("cant get directions: %v", err)
	}

	rawResponse, _ := json.Marshal(response)
	var planResponse models.PlanResponse
	json.Unmarshal(rawResponse, &planResponse)

	itinearies := planResponse.Plan.Itineraries

	if options.WalkReluctance >= 4 {
		sort.Slice(itinearies, func(i, j int) bool {
			return itinearies[i].WalkTime < itinearies[j].WalkTime
		})
	}

	return &models.PlanDataResponse{
		Itineraries: itinearies,
	}, nil
}
