package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// stolen from devops and other places
type GraphQLClient struct {
	GQLServer *url.URL
	Token     string
	Logger    log.Logger
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
	Data   *ShipReleaseResult `json:"data,omitempty"`
	Errors []GraphQLError     `json:"errors,omitempty"`
}

type ShipReleaseResult struct {
	PromoteResult map[string]interface{} `json:"promoteRelease"`
}

func (c *GraphQLClient) PromoteRelease(
	spec,
	channelId,
	semver,
	releaseNotes string,
) (*ShipReleaseResult, error) {
	debug := log.With(level.Debug(c.Logger), "type", "graphQLClient", "semver", semver)

	requestObj := GraphQLRequest{
		Query: `
mutation($channelId: ID!, $semver: String!, $spec: String!, $releaseNotes: String) {
      promoteRelease(
		channelId: $channelId
		semver: $semver
		spec: $spec
		releaseNotes: $releaseNotes
)
}`,
		Variables: map[string]string{
			"spec":         spec,
			"channelId":    channelId,
			"semver":       semver,
			"releaseNotes": releaseNotes,
		},
	}

	body, err := json.Marshal(requestObj)
	if err != nil {
		return nil, errors.Wrap(err, "marshal body")
	}

	bodyReader := bytes.NewReader(body)

	req, err := http.NewRequest("POST", c.GQLServer.String(), bodyReader)
	if err != nil {
		return nil, errors.Wrap(err, "marshal body")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "marshal body")
	}
	if resp == nil {
		return nil, errors.New("nil response from gql")
	}
	if resp.Body == nil {
		return nil, errors.New("nil response.Body from gql")
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	debug.Log("body", responseBody)
	if err != nil {
		return nil, errors.Wrap(err, "marshal body")
	}

	response := GraphQLResponse{}
	if err := json.Unmarshal(responseBody, &response); err != nil {

	}

	if response.Errors != nil && len(response.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range response.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return nil, multiErr.ErrorOrNil()
	}

	return response.Data, nil
}
