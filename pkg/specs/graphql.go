package specs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const getAppspecQuery = `
query ship($channel: String) {
  ship(channel: $channel) {
	spec
  }
}`

// GraphQLClient is a client for the graphql Payload API
type GraphQLClient struct {
	GQLServer *url.URL
}

// GraphQLRequest is a json-serializable request to the graphql server
type GraphQLRequest struct {
	Query         string            `json:"query"`
	Variables     map[string]string `json:"variables"`
	OperationName string            `json:"operationName"`
}

// GraphQLError represents an error returned by the graphql server
type GraphQLError struct {
	Locations []map[string]interface{} `json:"locations"`
	Message   string                   `json:"message"`
}

// GraphQLResponse is the top-level response object from the graphql server
type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLClientFromViper builds a new client using a viper instance
func GraphQLClientFromViper(v *viper.Viper) (*GraphQLClient, error) {
	addr := v.GetString("graphql_api_address")
	server, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse GQL server address %s", addr)
	}
	return &GraphQLClient{
		GQLServer: server,
	}, nil
}

// GetSpec gets an Payload payload from the graphql server
func (c *GraphQLClient) GetSpec(customerID, installationID string) (string, error) {
	requestObj := GraphQLRequest{
		Query: getAppspecQuery,
	}

	body, err := json.Marshal(requestObj)
	if err != nil {
		return "", errors.Wrap(err, "marshal request")
	}

	bodyReader := ioutil.NopCloser(bytes.NewReader(body))
	authString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", customerID, installationID)))

	graphQLRequest := &http.Request{
		URL: c.GQLServer,
		Header: map[string][]string{
			"Authorization": {"Basic " + authString},
			"Content-Type":  {"application/json"},
		},
		Method: http.MethodPost,
		Body:   bodyReader,
	}

	resp, err := http.DefaultClient.Do(graphQLRequest)
	if err != nil {
		return "", errors.Wrap(err, "send request")
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrap(err, "read body")
	}

	ship := GraphQLResponse{}

	if err := json.Unmarshal(responseBody, &ship); err != nil {
		return "", errors.Wrap(err, "unmarshal response")
	}

	if ship.Errors != nil && len(ship.Errors) > 0 {
		return "", errors.Wrap(errors.New(ship.Errors[0].Message), "graphql response")
	}

	return "", nil
}
