package web

import (
	"testing"

	"net/http"

	"context"

	"github.com/jarcoal/httpmock"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/templates"
	"github.com/replicatedhq/ship/pkg/testing/logger"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

type TestWebAsset struct {
	Name               string
	Asset              api.WebAsset
	ExpectFiles        map[string]interface{}
	ExpectedErr        error
	MockedRespBody     string
	RegisterResponders func()
}

func TestWebStep(t *testing.T) {
	tests := []TestWebAsset{
		{
			Name: "get",
			Asset: api.WebAsset{
				AssetShared: api.AssetShared{
					Dest: "asset.txt",
				},
				Headers: nil,
				URL:     "http://foo.bar",
			},
			RegisterResponders: func() {
				httpmock.RegisterResponder("GET", "http://foo.bar",
					httpmock.NewStringResponder(200, "hi from foo.bar"))
			},
			ExpectFiles: map[string]interface{}{
				"asset.txt": "hi from foo.bar",
			},
			ExpectedErr: nil,
		},
		{
			Name: "get error",
			Asset: api.WebAsset{
				AssetShared: api.AssetShared{
					Dest: "asset.txt",
				},
				Headers: nil,
				URL:     "http://foo.bar",
			},
			RegisterResponders: func() {
				httpmock.RegisterResponder("GET", "http://foo.bar",
					httpmock.NewStringResponder(500, "NOPE!"))
			},
			ExpectFiles: map[string]interface{}{},
			ExpectedErr: errors.New("Get web asset from http://foo.bar: received response with status 500"),
		},
		// {
		// 	Name: "post",
		// 	Asset: api.WebAsset{
		// 		AssetShared: api.AssetShared{
		// 			Dest: "asset.txt",
		// 		},
		// 		Body:    "stuff to post",
		// 		Headers: nil,
		// 		Method:  "POST",
		// 		URL:     "http://foo.bar",
		// 	},
		// 	RegisterResponders: func() {
		// 		httpmock.RegisterResponder("POST", "http://foo.bar",
		// 			httpmock.NewStringResponder(200, "hi from foo.bar"))
		// 	},
		// 	ExpectFiles: map[string]interface{}{
		// 		"asset.txt": "hi from foo.bar",
		// 	},
		// 	ExpectedErr: nil,
		// },
		// {
		// 	Name: "post error",
		// 	Asset: api.WebAsset{
		// 		AssetShared: api.AssetShared{
		// 			Dest: "asset.txt",
		// 		},
		// 		Body:    "stuff to post",
		// 		Headers: nil,
		// 		Method:  "POST",
		// 		URL:     "http://foo.bar",
		// 	},
		// 	RegisterResponders: func() {
		// 		httpmock.RegisterResponder("POST", "http://foo.bar",
		// 			httpmock.NewStringResponder(500, "NOPE!"))
		// 	},
		// 	ExpectFiles: map[string]interface{}{},
		// 	ExpectedErr: errors.New("Get web asset from http://foo.bar: received response with status 500"),
		// },
		// {
		// 	Name: "headers",
		// 	Asset: api.WebAsset{
		// 		AssetShared: api.AssetShared{
		// 			Dest: "asset.txt",
		// 		},
		// 		Body: "some stuff to post",
		// 		Headers: map[string][]string{
		// 			"Authorization": {"my auth"},
		// 		},
		// 		Method: "POST",
		// 		URL:    "http://foo.bar",
		// 	},
		// 	RegisterResponders: func() {
		// 		httpmock.RegisterResponder("POST", "http://foo.bar",
		// 			func(req *http.Request) (*http.Response, error) {
		// 				header := req.Header.Get("Authorization")
		//
		// 				decoded, _ := base64.StdEncoding.DecodeString(header)
		// 				if string(decoded) != "my auth" {
		// 					return httpmock.NewStringResponse(500, "mock headers != test headers"), nil
		// 				}
		//
		// 				return httpmock.NewStringResponse(200, "hi from foo.bar"), nil
		// 			})
		// 	},
		// 	ExpectFiles: map[string]interface{}{
		// 		"asset.txt": "hi from foo.bar",
		// 	},
		// 	ExpectedErr: nil,
		// },
		// {
		// 	Name: "headers error",
		// 	Asset: api.WebAsset{
		// 		AssetShared: api.AssetShared{
		// 			Dest: "asset.txt",
		// 		},
		// 		Body: "some stuff to post",
		// 		Headers: map[string][]string{
		// 			"Authorization": {"my auth"},
		// 		},
		// 		Method: "POST",
		// 		URL:    "http://foo.bar",
		// 	},
		// 	RegisterResponders: func() {
		// 		httpmock.RegisterResponder("POST", "http://foo.bar",
		// 			func(req *http.Request) (*http.Response, error) {
		// 				header := req.Header.Get("Authorization")
		//
		// 				decoded, _ := base64.StdEncoding.DecodeString(header)
		// 				if string(decoded) != "NOT my auth" {
		// 					return httpmock.NewStringResponse(500, "mock headers != test headers"), nil
		// 				}
		//
		// 				return httpmock.NewStringResponse(200, "hi from foo.bar"), nil
		// 			})
		// 	},
		// 	ExpectFiles: map[string]interface{}{},
		// 	ExpectedErr: errors.New("Get web asset from http://foo.bar: received response with status 500"),
		// },
		// {
		// 	Name: "advanced post with headers",
		// 	Asset: api.WebAsset{
		// 		AssetShared: api.AssetShared{
		// 			Dest: "asset.txt",
		// 		},
		// 		Body: "some stuff to post",
		// 		Headers: map[string][]string{
		// 			"Authorization": {"my auth"},
		// 		},
		// 		Method: "POST",
		// 		URL:    "http://foo.bar",
		// 	},
		// 	RegisterResponders: func() {
		// 		httpmock.RegisterResponder("POST", "http://foo.bar",
		// 			func(req *http.Request) (*http.Response, error) {
		// 				header := req.Header.Get("Authorization")
		//
		// 				decoded, _ := base64.StdEncoding.DecodeString(header)
		// 				if string(decoded) != "my auth" {
		// 					return httpmock.NewStringResponse(500, "mock headers != test headers"), nil
		// 				}
		//
		// 				body, _ := ioutil.ReadAll(req.Body)
		// 				if string(body) != "some stuff to post" {
		// 					return httpmock.NewStringResponse(500, "mock body != test body"), nil
		// 				}
		//
		// 				resp, err := httpmock.NewJsonResponse(200, "some stuff to post")
		// 				if err != nil {
		// 					return httpmock.NewStringResponse(500, "NOPE!"), nil
		// 				}
		// 				return resp, nil
		// 			})
		// 	},
		// 	ExpectFiles: map[string]interface{}{},
		// 	ExpectedErr: errors.New("Get web asset from http://foo.bar: received response with status 500"),
		// },
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			req := require.New(t)

			v := viper.New()
			testLogger := &logger.TestLogger{T: t}
			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			client := &http.Client{}

			step := &DefaultStep{
				Logger: testLogger,
				Fs:     fs,
				Viper:  v,
				BuilderBuilder: &templates.BuilderBuilder{
					Logger: testLogger,
					Viper:  v,
				},
				Client: client,
			}

			httpmock.Activate()
			defer httpmock.DeactivateAndReset()
			test.RegisterResponders()

			err := step.Execute(
				test.Asset,
				api.ReleaseMetadata{},
				[]libyaml.ConfigGroup{},
				map[string]interface{}{},
			)(context.Background())

			if test.ExpectedErr == nil {
				req.NoError(err)
			} else {
				req.Error(err, "expected error "+test.ExpectedErr.Error())
				req.Equal(test.ExpectedErr.Error(), err.Error())
			}

			for name, contents := range test.ExpectFiles {
				actual, err := fs.ReadFile(name)
				req.NoError(err)
				req.Equal(contents, string(actual))
			}

		})
	}
}
