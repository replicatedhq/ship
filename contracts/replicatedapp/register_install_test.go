package replicatedapp

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	replapp "github.com/replicatedhq/ship/pkg/specs/replicatedapp"
)

func Test_RegisterInstall(t *testing.T) {
	customerID := "ship-register-customer-0"
	installationID := "ship-register-installation-0"
	channelID := "ship-register-channel-0"
	releaseID := "ship-register-release-0"
	var test = func() (err error) {
		req := require.New(t)

		v := viper.New()
		v.Set("customer-endpoint", fmt.Sprintf("http://localhost:%d/graphql", pact.Server.Port))

		gqlClient, err := replapp.NewGraphqlClient(v, http.DefaultClient)
		req.NoError(err)

		err = gqlClient.RegisterInstall(customerID, installationID, channelID, releaseID)
		req.NoError(err)

		return nil
	}

	pact.AddInteraction().
		Given("A request to register an install").
		UponReceiving("A request to register an install").
		WithRequest(dsl.Request{
			Method: "POST",
			Path:   dsl.String("/graphql"),
			Headers: dsl.MapMatcher{
				"Authorization": dsl.String(fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", customerID, installationID))))),
				"Content-Type":  dsl.String("application/json"),
			},
			Body: map[string]interface{}{
				"operationName": "",
				"query":         replapp.RegisterInstallQuery,
				"variables": map[string]interface{}{
					"channelId": channelID,
					"releaseId": releaseID,
				},
			},
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
			Body: map[string]interface{}{
				"data": map[string]interface{}{
					"shipRegisterInstall": dsl.Like(true),
				},
			},
		})

	if err := pact.Verify(test); err != nil {
		t.Fatalf("Error on Verify: %v", err)
	}
}
