package utils

import (
	"context"
	"fmt"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/redis/go-redis/v9"
	"github.com/uber/h3-go/v4"
)

const H3_RESOLUTION = 8

func ToCell(coor models.Coordinate) *h3.Cell {
	cell := h3.LatLngToCell(h3.LatLng{
		Lat: coor.Latitude,
		Lng: coor.Longitude,
	}, H3_RESOLUTION)

	return &cell
}

func KRingSearch(origin h3.LatLng, diskDist int, client *redis.Client) [][]string {
	cellOrigin := h3.LatLngToCell(origin, H3_RESOLUTION)

	disk := h3.GridDiskDistances(cellOrigin, diskDist)
	diskKeys := make([][]string, 0)
	for _, dist := range disk {
		keys := make([]string, 0)

		for _, cell := range dist {
			key := fmt.Sprintf("h3:%s", cell.String())
			keys = append(keys, key)
		}
		diskKeys = append(diskKeys, keys)
	}

	nearestProperties := make([][]string, 0)
	ctx := context.Background()

	for i, keys := range diskKeys {
		if i == 0 {
			continue
		}

		for _, cellKey := range keys {
			properties := client.LRange(ctx, cellKey, 0, -1)
			if len(properties.Val()) != 0 {
				nearestProperties = append(nearestProperties, properties.Val())
			}
		}
	}

	return nearestProperties
}

func RadiusSearch(origin h3.LatLng, radius float64, client *redis.Client) []redis.GeoLocation {
	ctx := context.Background()
	properties := client.GeoSearchLocation(ctx, "properties", &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  origin.Lng,
			Latitude:   origin.Lat,
			Radius:     radius,
			RadiusUnit: "km",
			Sort:       "ASC",
		},
		WithDist: true,
		WithHash: true,
	})

	return properties.Val()
}
