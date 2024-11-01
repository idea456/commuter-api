package transport

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/idea456/commuter-api/pkg/models"
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	conn *redis.Client
}

var redisClient *RedisClient

func NewRedisClient() (*RedisClient, error) {
	uri := os.Getenv("REDIS_URL")
	if uri == "" {
		return nil, errors.New("Redis URL not specified in environment variable")
	}
	if redisClient == nil {
		options := &redis.Options{
			Addr:     uri,
			Password: "",
			DB:       0,
		}
		client := redis.NewClient(options)

		redisClient = &RedisClient{
			conn: client,
		}

		log.Println("Connected to Redis.")
	}

	return redisClient, nil
}

func (client *RedisClient) Disconnect() {
	if err := client.conn.Close(); err != nil {
		panic(err)
	}
}

func (client *RedisClient) GeoAdd(key string, origin models.Coordinate) {
	client.conn.GeoAdd(context.TODO(), "properties", &redis.GeoLocation{
		Name:      key,
		Longitude: origin.Longitude,
		Latitude:  origin.Latitude,
	})
}

func (client *RedisClient) RadiusSearch(origin models.Coordinate, radius float64) []redis.GeoLocation {
	properties := client.conn.GeoSearchLocation(context.TODO(), "properties", &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Longitude:  origin.Longitude,
			Latitude:   origin.Latitude,
			Radius:     radius,
			RadiusUnit: "km",
			Sort:       "ASC",
		},
		WithDist: true,
	})

	return properties.Val()
}

func (client *RedisClient) ReadStream(consumer *chan interface{}) {

	streams, err := client.conn.XRead(context.TODO(), &redis.XReadArgs{
		Streams: []string{"search:trx", "$"},
		Block:   0,
	}).Result()
	if err != nil {
		log.Fatal(err)
	}

	for _, stream := range streams {
		for _, message := range stream.Messages {
			*consumer <- message
		}
	}
}

func (client *RedisClient) ListPush(key string, values ...interface{}) {
	client.conn.LPush(context.TODO(), key, values)
}

func (client *RedisClient) FlushAll() {
	client.conn.FlushAll(context.TODO())
}
