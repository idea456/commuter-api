package transport

import (
	"context"

	"github.com/machinebox/graphql"
)


type GraphQLClient struct {
	conn *graphql.Client
}

func NewGraphQLClient(graphQLURL string) (*GraphQLClient, error) {
	client := graphql.NewClient(graphQLURL)
	graphqlClient := &GraphQLClient{
		conn: client,
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
