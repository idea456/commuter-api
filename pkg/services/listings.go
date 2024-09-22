package services

import (
	"log"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/idea456/commuter-api/pkg/transport"
)

type ListingService struct {
	MongoDBClient *transport.MongoDBClient
}

func NewListingService() *ListingService {
	client, err := transport.NewMongoDBClient()
	if err != nil {
		log.Fatal(err)
	}

	return &ListingService{
		MongoDBClient: client,
	}
}

func (svc *ListingService) GetListings(propertyId string) []models.Listing {
	return []models.Listing{}
}

func (svc *ListingService) GetListing(listingId string) models.Listing {
	return models.Listing{}
}
