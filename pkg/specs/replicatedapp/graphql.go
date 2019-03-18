package replicatedapp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/spf13/viper"
)

const getAppspecQuery = `
query($semver: String) {
  shipRelease (semver: $semver) {
    id
    channelId
    channelName
    channelIcon
    semver
    releaseNotes
    spec
    images {
      url
      source
      appSlug
      imageKey
    }
    githubContents {
      repo
      path
      ref
      files {
        name
        path
        sha
        size
        data
      }
    }
    created
    registrySecret
  }
}`

const getSlugAppSpecQuery = `
query($appSlug: String!, $licenseID: String, $releaseID: String, $semver: String) {
  shipSlugRelease (appSlug: $appSlug, licenseID: $licenseID, releaseID: $releaseID, semver: $semver) {
    id
    channelId
    channelName
    channelIcon
    semver
    releaseNotes
    spec
    images {
      url
      source
      appSlug
      imageKey
    }
    githubContents {
      repo
      path
      ref
      files {
        name
        path
        sha
        size
        data
      }
    }
    created
    registrySecret
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

// GQLGetReleaseResponse is the top-level response object from the graphql server
type GQLGetReleaseResponse struct {
	Data   ShipReleaseWrapper `json:"data,omitempty"`
	Errors []GraphQLError     `json:"errors,omitempty"`
}

// GQLGetSlugReleaseResponse is the top-level response object from the graphql server
type GQLGetSlugReleaseResponse struct {
	Data   ShipSlugReleaseWrapper `json:"data,omitempty"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

// ShipReleaseWrapper wraps the release response form GQL
type ShipReleaseWrapper struct {
	ShipRelease ShipRelease `json:"shipRelease"`
}

// ShipSlugReleaseWrapper wraps the release response form GQL
type ShipSlugReleaseWrapper struct {
	ShipSlugRelease ShipRelease `json:"shipSlugRelease"`
}

type Image struct {
	URL      string `json:"url"`
	Source   string `json:"source"`
	AppSlug  string `json:"appSlug"`
	ImageKey string `json:"imageKey"`
}

type GithubFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Sha  string `json:"sha"`
	Size int64  `json:"size"`
	Data string `json:"data"`
}

type GithubContent struct {
	Repo  string       `json:"repo"`
	Path  string       `json:"path"`
	Ref   string       `json:"ref"`
	Files []GithubFile `json:"files"`
}

// ShipRelease is the release response form GQL
type ShipRelease struct {
	ID             string           `json:"id"`
	ChannelID      string           `json:"channelId"`
	ChannelName    string           `json:"channelName"`
	ChannelIcon    string           `json:"channelIcon"`
	Semver         string           `json:"semver"`
	ReleaseNotes   string           `json:"releaseNotes"`
	Spec           string           `json:"spec"`
	Images         []Image          `json:"images"`
	GithubContents []GithubContent  `json:"githubContents"`
	Created        string           `json:"created"` // TODO: this time is not in RFC 3339 format
	RegistrySecret string           `json:"registrySecret"`
	Entitlements   api.Entitlements `json:"entitlements"`
}

// GQLRegisterInstallResponse is the top-level response object from the graphql server
type GQLRegisterInstallResponse struct {
	Data struct {
		ShipRegisterInstall bool `json:"shipRegisterInstall"`
	} `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type callInfo struct {
	username string
	password string
	request  GraphQLRequest
	upstream string
}

// ToReleaseMeta linter
func (r *ShipRelease) ToReleaseMeta() api.ReleaseMetadata {
	return api.ReleaseMetadata{
		ReleaseID:      r.ID,
		ChannelID:      r.ChannelID,
		ChannelName:    r.ChannelName,
		ChannelIcon:    r.ChannelIcon,
		Semver:         r.Semver,
		ReleaseNotes:   r.ReleaseNotes,
		Created:        r.Created,
		RegistrySecret: r.RegistrySecret,
		Images:         r.apiImages(),
		GithubContents: r.githubContents(),
		Entitlements:   r.Entitlements,
	}
}

func (r *ShipRelease) apiImages() []api.Image {
	result := []api.Image{}
	for _, image := range r.Images {
		result = append(result, api.Image(image))
	}
	return result
}

func (r *ShipRelease) githubContents() []api.GithubContent {
	result := []api.GithubContent{}
	for _, content := range r.GithubContents {
		files := []api.GithubFile{}
		for _, file := range content.Files {
			files = append(files, api.GithubFile(file))
		}
		apiCont := api.GithubContent{
			Repo:  content.Repo,
			Path:  content.Path,
			Ref:   content.Ref,
			Files: files,
		}
		result = append(result, apiCont)
	}
	return result
}

// NewGraphqlClient builds a new client using a viper instance
func NewGraphqlClient(v *viper.Viper, client *http.Client) (*GraphQLClient, error) {
	addr := v.GetString("customer-endpoint")
	server, err := url.ParseRequestURI(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse GQL server address %s", addr)
	}
	return &GraphQLClient{
		GQLServer: server,
		Client:    client,
	}, nil
}

// GetRelease gets a payload from the graphql server
func (c *GraphQLClient) GetRelease(selector *Selector) (*ShipRelease, error) {
	requestObj := GraphQLRequest{
		Query: getAppspecQuery,
		Variables: map[string]string{
			"semver": selector.ReleaseSemver,
		},
	}

	ci := callInfo{
		username: selector.GetBasicAuthUsername(),
		password: selector.InstallationID,
		request:  requestObj,
		upstream: selector.Upstream,
	}

	shipResponse := &GQLGetReleaseResponse{}
	if err := c.callGQL(ci, shipResponse); err != nil {
		return nil, err
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

// GetSlugRelease gets a release from the graphql server by app slug
func (c *GraphQLClient) GetSlugRelease(selector *Selector) (*ShipRelease, error) {
	requestObj := GraphQLRequest{
		Query: getSlugAppSpecQuery,
		Variables: map[string]string{
			"appSlug":   selector.AppSlug,
			"licenseID": selector.LicenseID,
			"releaseID": selector.ReleaseID,
			"semver":    selector.ReleaseSemver,
		},
	}

	ci := callInfo{
		username: selector.GetBasicAuthUsername(),
		password: selector.InstallationID,
		request:  requestObj,
		upstream: selector.Upstream,
	}

	shipResponse := &GQLGetSlugReleaseResponse{}
	if err := c.callGQL(ci, shipResponse); err != nil {
		return nil, err
	}

	if shipResponse.Errors != nil && len(shipResponse.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range shipResponse.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return nil, multiErr.ErrorOrNil()
	}

	return &shipResponse.Data.ShipSlugRelease, nil
}

func (c *GraphQLClient) RegisterInstall(customerID, installationID, channelID, releaseID string) error {
	requestObj := GraphQLRequest{
		Query: `
mutation($channelId: String!, $releaseId: String!) {
  shipRegisterInstall(
    channelId: $channelId
    releaseId: $releaseId
  )
}`,
		Variables: map[string]string{
			"channelId": channelID,
			"releaseId": releaseID,
		},
	}

	ci := callInfo{
		username: customerID,
		password: installationID,
		request:  requestObj,
	}

	shipResponse := &GQLRegisterInstallResponse{}
	if err := c.callGQL(ci, shipResponse); err != nil {
		return err
	}

	if shipResponse.Errors != nil && len(shipResponse.Errors) > 0 {
		var multiErr *multierror.Error
		for _, err := range shipResponse.Errors {
			multiErr = multierror.Append(multiErr, fmt.Errorf("%s: %s", err.Code, err.Message))

		}
		return multiErr.ErrorOrNil()
	}

	return nil
}

func (c *GraphQLClient) callGQL(ci callInfo, result interface{}) error {
	body, err := json.Marshal(ci.request)
	if err != nil {
		return errors.Wrap(err, "marshal request")
	}

	bodyReader := ioutil.NopCloser(bytes.NewReader(body))

	gqlServer := c.GQLServer.String()
	if ci.upstream != "" {
		gqlServer = ci.upstream
	}
	graphQLRequest, err := http.NewRequest(http.MethodPost, gqlServer, bodyReader)
	if err != nil {
		return errors.Wrap(err, "create new request")
	}

	graphQLRequest.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}

	if ci.username != "" || ci.password != "" {
		authString := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", ci.username, ci.password)))
		graphQLRequest.Header["Authorization"] = []string{"Basic " + authString}
	}

	resp, err := c.Client.Do(graphQLRequest)
	if err != nil {
		return errors.Wrap(err, "send request")
	}

	responseBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "read body")
	}

	if err := json.Unmarshal(responseBody, result); err != nil {
		return errors.Wrapf(err, "unmarshal response %s", responseBody)
	}

	return nil
}
