package e2e

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/stretchr/testify/require"
)

// stolen from devops and other places
type GraphQLClient struct {
	GQLServer *url.URL
	Token     string
	assert    *require.Assertions
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
	Data   *ShipReleaseResult `json:"data,omitempty"`
	Errors []GraphQLError     `json:"errors,omitempty"`
}

type ShipReleaseResult struct {
	PromoteResult map[string]interface{} `json:"promoteRelease"`
}

func (c *GraphQLClient) promoteRelease(spec, channelId, semver string) {
	requestObj := GraphQLRequest{
		Query: `
mutation($channelId: ID!, $semver: String!, $spec: String!) {
      promoteRelease(
		channelId: $channelId
		releaseNotes: "Integration test run on ` + time.Now().String() + `"
		semver: $semver
		spec: $spec
) {
	id
  }
}`,
		Variables: map[string]string{
			"spec":      spec,
			"channelId": channelId,
			"semver":    semver,
		},
	}

	body, err := json.Marshal(requestObj)
	c.assert.NoError(err)

	bodyReader := bytes.NewReader(body)

	req, err := http.NewRequest("POST", c.GQLServer.String(), bodyReader)
	c.assert.NoError(err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	c.assert.NoError(err)
	c.assert.NotNil(resp)
	c.assert.NotNil(resp.Body)

	responseBody, err := ioutil.ReadAll(resp.Body)
	c.assert.NoError(err)

	response := GraphQLResponse{}
	c.assert.NoError(json.Unmarshal(responseBody, &response))
	c.assert.Empty(response.Errors)
}
