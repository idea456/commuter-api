package transport

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBClient struct {
	conn *mongo.Client
}

var mongoDBClient *MongoDBClient

func NewMongoDBClient() (*MongoDBClient, error) {
	if mongoDBClient == nil {
		if err := godotenv.Load(); err != nil {
			return nil, err
		}

		uri := os.Getenv("MONGODB_URI")
		if uri == "" {
			return nil, errors.New("MongoDB URI not specified in environment variable")
		}

		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
		if err != nil {
			return nil, err
		}
		mongoDBClient = &MongoDBClient{
			conn: client,
		}
		log.Println("Connected to MongoDB")
	}

	return mongoDBClient, nil
}

func (client *MongoDBClient) Disconnect() {
	if err := client.conn.Disconnect(context.TODO()); err != nil {
		panic(err)
	}
}

func (client *MongoDBClient) FindOne(databaseName string, collectionName string, filter bson.M) *mongo.SingleResult {
	coll := client.conn.Database(databaseName).Collection(collectionName)

	result := coll.FindOne(context.TODO(), filter)
	return result
}

func (client *MongoDBClient) Find(databaseName string, collectionName string, filter bson.M) (*mongo.Cursor, error) {
	coll := client.conn.Database(databaseName).Collection(collectionName)

	cursor, err := coll.Find(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	return cursor, nil
}

func (client *MongoDBClient) InsertItem(databaseName string, collectionName string, value interface{}) error {
	coll := client.conn.Database(databaseName).Collection(collectionName)

	_, err := coll.InsertOne(context.TODO(), value)
	if err != nil {
		return err
	}

	return nil
}

func (client *MongoDBClient) DropDatabase(databaseName string) error {
	err := client.conn.Database(databaseName).Drop(context.TODO())
	return err
}
