package specs

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

const getAppspecQuery = `
query {
  shipRelease {
    id
    channelId
    channelName
    semver
    releaseNotes
    spec
    created
  }
}`

// GraphQLClient is a client for the graphql Payload API
type GraphQLClient struct {
	GQLServer *url.URL
	Client    *http.Client
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
	Code      string                   `json:"code"`
}

// GraphQLResponse is the top-level response object from the graphql server
type GraphQLResponse struct {
	Data   ShipReleaseWrapper `json:"data,omitempty"`
	Errors []GraphQLError     `json:"errors,omitempty"`
}

// ShipReleaseWrapper wraps the release response form GQL
type ShipReleaseWrapper struct {
	ShipRelease ShipRelease `json:"shipRelease"`
}

// ShipRelease is the release response form GQL
type ShipRelease struct {
	ID           string `json:"id"`
	ChannelID    string `json:"channelId"`
	ChannelName  string `json:"channelName"`
	Semver       string `json:"semver"`
	ReleaseNotes string `json:"releaseNotes"`
	Spec         string `json:"spec"`
	Created      string `json:"created"`
}

// GraphQLClientFromViper builds a new client using a viper instance
func GraphQLClientFromViper(v *viper.Viper) (*GraphQLClient, error) {
	addr := v.GetString("customer-endpoint")
	server, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse GQL server address %s", addr)
	}
	return &GraphQLClient{
		GQLServer: server,
		Client:    http.DefaultClient,
	}, nil
}

// GetRelease gets a payload from the graphql server
func (c *GraphQLClient) GetRelease(customerID, installationID string) (*ShipRelease, error) {
	requestObj := GraphQLRequest{
		Query: getAppspecQuery,
	}

	body, err := json.Marshal(requestObj)
	if err != nil {
		return nil, errors.Wrap(err, "marshal request")
	}

	bodyReader := ioutil.NopCloser(bytes.NewReader(body))
	authString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", customerID, installationID)))

	graphQLRequest, err := http.NewRequest(http.MethodPost, c.GQLServer.String(), bodyReader)

	graphQLRequest.Header = map[string][]string{
		"Authorization": {"Basic " + authString},
		"Content-Type":  {"application/json"},
	}

	resp, err := c.Client.Do(graphQLRequest)
	if err != nil {
		return nil, errors.Wrap(err, "send request")
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read body")
	}

	shipResponse := GraphQLResponse{}

	if err := json.Unmarshal(responseBody, &shipResponse); err != nil {
		return nil, errors.Wrapf(err, "unmarshal response %s", responseBody)
	}

	if shipResponse.Errors != nil && len(shipResponse.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range shipResponse.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return nil, multiErr.ErrorOrNil()
	}

	return &shipResponse.Data.ShipRelease, nil
}
