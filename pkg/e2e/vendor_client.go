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

// GraphQLResponsePromoteRelease is the top-level response object from the graphql server
type GraphQLResponsePromoteRelease struct {
	Data   *ShipReleaseResult `json:"data,omitempty"`
	Errors []GraphQLError     `json:"errors,omitempty"`
}

type ShipReleaseResult struct {
	PromoteResult map[string]interface{} `json:"promoteRelease"`
}

// GraphQLResponsePromoteRelease is the top-level response object from the graphql server
type GraphQLResponseCreateChannel struct {
	Data   *ShipCreateChannelResult `json:"data,omitempty"`
	Errors []GraphQLError           `json:"errors,omitempty"`
}

type ShipCustomer struct {
	InstallationID string `json:"installationId"`
	ID             string `json:"id"`
}

type ShipChannel struct {
	Name      string         `json:"name"`
	ID        string         `json:"id"`
	Customers []ShipCustomer `json:"customers"`
}

type ShipCreateChannelResult struct {
	CreateChannel ShipChannel `json:"createChannel"`
}

type ShipAssignCustomerResult struct {
	AssignCustomerToChannel ShipChannel `json:"assignCustomerToChannel"`
}

type ShipChannelListResult struct {
	SearchChannels []ShipChannel `json:"searchChannels"`
}

type GraphQLResponseListChannel struct {
	Data   *ShipChannelListResult `json:"data,omitempty"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

type GraphQLResponseAssignCustomer struct {
	Data   *ShipAssignCustomerResult `json:"data,omitempty"`
	Errors []GraphQLError            `json:"errors,omitempty"`
}

func (r GraphQLResponseListChannel) GraphQLError() []GraphQLError {
	return r.Errors
}
func (r GraphQLResponseCreateChannel) GraphQLError() []GraphQLError {
	return r.Errors
}
func (r GraphQLResponsePromoteRelease) GraphQLError() []GraphQLError {
	return r.Errors
}

func (r GraphQLResponseAssignCustomer) GraphQLError() []GraphQLError {
	return r.Errors
}

type Errer interface {
	GraphQLError() []GraphQLError
}

func (c *GraphQLClient) GetOrCreateChannel(name string) (*ShipChannel, error) {
	requestObj := GraphQLRequest{
		Query: `
query($channelName: String!) {
  searchChannels(channelName: $channelName) {
    id name
  }
}`,
		Variables: map[string]string{"channelName": name},
	}
	response := GraphQLResponseListChannel{}
	err := c.executeRequest(requestObj, &response)
	if err != nil {
		return nil, errors.Wrapf(err, "execute request")
	}

	if err := c.checkErrors(response); err != nil {
		return nil, err
	}

	if response.Data != nil && len(response.Data.SearchChannels) != 0 {
		return &response.Data.SearchChannels[0], nil
	}

	channel, err := c.CreateChannel(name)
	return channel, errors.Wrap(err, "create channel")
}

func (c *GraphQLClient) PromoteRelease(
	spec,
	channelID,
	semver,
	releaseNotes string,
) (*ShipReleaseResult, error) {

	requestObj := GraphQLRequest{
		Query: `
mutation($channelId: ID!, $semver: String!, $spec: String!, $releaseNotes: String) {
      promoteRelease(
		channelId: $channelId
		semver: $semver
		spec: $spec
		releaseNotes: $releaseNotes
) {
  id }
}`,
		Variables: map[string]string{
			"spec":         spec,
			"channelId":    channelID,
			"semver":       semver,
			"releaseNotes": releaseNotes,
		},
	}
	response := GraphQLResponsePromoteRelease{}
	err := c.executeRequest(requestObj, &response)
	if err != nil {
		return nil, errors.Wrapf(err, "execute request")
	}
	if err := c.checkErrors(response); err != nil {
		return nil, err
	}

	return response.Data, nil
}

func (c *GraphQLClient) CreateChannel(name string) (*ShipChannel, error) {
	requestObj := GraphQLRequest{
		Query: `
mutation($channelName: String!) {
  createChannel(channelName: $channelName) {
    id
    name
  }
}`,
		Variables: map[string]string{
			"channelName": name,
		},
	}
	response := GraphQLResponseCreateChannel{}
	err := c.executeRequest(requestObj, &response)
	if err != nil {
		return nil, errors.Wrapf(err, "execute request")
	}
	if err := c.checkErrors(response); err != nil {
		return nil, err
	}

	return &response.Data.CreateChannel, nil
}

func (c *GraphQLClient) EnsureCustomerOnChannel(customerId string, channelId string) (string, error) {
	requestObj := GraphQLRequest{
		Query: `
mutation($customerId: ID!, $channelId: ID!) {
  assignCustomerToChannel(customerId: $customerId, channelId: $channelId) {
    id
    name
	customers {
      id
      installationId
    }
  }
}`,
		Variables: map[string]string{
			"customerId": customerId,
			"channelId":  channelId,
		},
	}
	response := GraphQLResponseAssignCustomer{}
	err := c.executeRequest(requestObj, &response)
	if err != nil {
		return "", errors.Wrapf(err, "execute request")
	}
	if err := c.checkErrors(response); err != nil {
		return "", err
	}

	for _, customer := range response.Data.AssignCustomerToChannel.Customers {
		if customer.ID == customerId {
			return customer.InstallationID, nil
		}

	}
	return "", errors.Errorf("no matching customers returned when assigning customer %s to channel %s", customerId, channelId)

}

func (c *GraphQLClient) executeRequest(
	requestObj GraphQLRequest,
	deserializeTarget interface{},
) error {
	debug := log.With(level.Debug(c.Logger), "type", "graphQLClient", "semver")
	body, err := json.Marshal(requestObj)
	if err != nil {
		return errors.Wrap(err, "marshal body")
	}
	bodyReader := bytes.NewReader(body)
	req, err := http.NewRequest("POST", c.GQLServer.String(), bodyReader)
	if err != nil {
		return errors.Wrap(err, "marshal body")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Token)
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "marshal body")
	}
	if resp == nil {
		return errors.New("nil response from gql")
	}
	if resp.Body == nil {
		return errors.New("nil response.Body from gql")
	}
	responseBody, err := ioutil.ReadAll(resp.Body)
	debug.Log("body", responseBody)
	if err != nil {
		return errors.Wrap(err, "marshal body")
	}
	if err := json.Unmarshal(responseBody, deserializeTarget); err != nil {
		return errors.Wrap(err, "unmarshal response")
	}

	return nil
}

func (c *GraphQLClient) checkErrors(errer Errer) error {
	if errer.GraphQLError() != nil && len(errer.GraphQLError()) > 0 {
		var multiErr *multierror.Error
		for _, err := range errer.GraphQLError() {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))
		}
		if multiErr == nil {
			return fmt.Errorf("expected %d gql errors but none found", len(errer.GraphQLError()))
		}
		return multiErr.ErrorOrNil()
	}
	return nil
}
