package transport

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var dynamoDBClient *DynamoDBClient

type DynamoDBClient struct {
	sess *session.Session
	conn *dynamodb.DynamoDB
}

func initSession() *session.Session {
	newSess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("ap-southeast-1"),
	}))

	return newSess
}

func NewDynamoDBClient() *DynamoDBClient {
	if dynamoDBClient == nil {
		sess := initSession()
		dynamoDBClient = &DynamoDBClient{
			sess: sess,
			conn: dynamodb.New(sess),
		}
	}

	return dynamoDBClient
}

func (client *DynamoDBClient) PutItem(tableName string, value interface{}) (*dynamodb.PutItemOutput, error) {
	item, err := dynamodbattribute.MarshalMap(value)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item:      item,
	}
	result, err := client.conn.PutItem(input)
	if err != nil {
		return nil, err
	}

	return result, nil
}
