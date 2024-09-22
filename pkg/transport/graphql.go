package transport

import (
	"context"
	"errors"
	"os"

	"github.com/machinebox/graphql"
)

var graphqlClient *GraphQLClient

type GraphQLClient struct {
	conn *graphql.Client
}

func NewGraphQLClient() (*GraphQLClient, error) {
	url := os.Getenv("OTP_GRAPHQL_URL")
	if url == "" {
		return nil, errors.New("GraphQL URL not specified in environment variable")
	}
	if graphqlClient == nil {
		client := graphql.NewClient(url)
		graphqlClient = &GraphQLClient{
			conn: client,
		}
	}
	return graphqlClient, nil
}

func (client *GraphQLClient) Query(gql string) (interface{}, error) {
	req := graphql.NewRequest(gql)

	var responseData interface{}
	if err := client.conn.Run(context.TODO(), req, &responseData); err != nil {
		return nil, err
	}

	return responseData, nil
}
