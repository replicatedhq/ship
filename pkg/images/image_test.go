package images

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/testing/logger"
)

func TestResolveImageName(t *testing.T) {
	type oneResult struct {
		Name  string
		Tag   string
		IsErr bool
	}
	type testcase struct {
		Name     string
		ImageURL string
		Expect   oneResult
	}

	cases := []testcase{
		{
			Name:     "just image name",
			ImageURL: "redis",
			Expect: oneResult{
				Name:  "redis",
				Tag:   "latest",
				IsErr: false,
			},
		},
		{
			Name:     "image name and tag",
			ImageURL: "redis:7",
			Expect: oneResult{
				Name:  "redis",
				Tag:   "7",
				IsErr: false,
			},
		},
		{
			Name:     "image name, org, and tag",
			ImageURL: "awesome/redis:1.3",
			Expect: oneResult{
				Name:  "redis",
				Tag:   "1.3",
				IsErr: false,
			},
		},
		{
			Name:     "host, image name, org, and tag",
			ImageURL: "quay.io/awesome/redis:3.5",
			Expect: oneResult{
				Name:  "redis",
				Tag:   "3.5",
				IsErr: false,
			},
		},
		{
			Name:     "just some url",
			ImageURL: "https://www.google.com",
			Expect: oneResult{
				Name:  "",
				Tag:   "",
				IsErr: true,
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			imageName, imageTag, err := resolveImageName(test.ImageURL)
			if test.Expect.IsErr {
				require.New(t).Error(err)
			} else {
				require.New(t).NoError(err)
				require.New(t).Equal(test.Expect.Name, imageName)
				require.New(t).Equal(test.Expect.Tag, imageTag)
			}
		})
	}
}

func TestResolvePullUrl(t *testing.T) {
	type testcase struct {
		Name      string
		Asset     api.DockerAsset
		ExpectURL string
	}
	cases := []testcase{
		{
			Name: "replicated private image",
			Asset: api.DockerAsset{
				Image:  "registry.replicated.com/library/retraced-api:1.1.12-slim-20180329",
				Source: "replicated",
			},
			ExpectURL: "registry.replicated.com/library/retraced-api:1.1.12-slim-20180329",
		},
		{
			Name: "public image with host name",
			Asset: api.DockerAsset{
				Image:  "quay.io/awesome/redis:1.1",
				Source: "public",
			},
			ExpectURL: "quay.io/awesome/redis:1.1",
		},
		{
			Name: "private proxied image without host name",
			Asset: api.DockerAsset{
				Image:  "replicated/www:3",
				Source: "dockerhub",
			},
			ExpectURL: fmt.Sprintf("%s/awesomeapp/jjzpr9u62gaz2.www:3", replicatedRegistry()),
		},
		{
			Name: "private proxied image with host name",
			Asset: api.DockerAsset{
				Image:  "quay.io/redacted/chatops:f3c689e",
				Source: "quayio",
			},
			ExpectURL: fmt.Sprintf("%s/awesomeapp/jjzpr9u62gaz4.chatops:f3c689e", replicatedRegistry()),
		},
		{
			Name: "private proxied image with no slug",
			Asset: api.DockerAsset{
				Image:  "quay.io/redacted/hugops:f3c689e",
				Source: "quayio",
			},
			ExpectURL: fmt.Sprintf("%s/ship/jjzpr9u62gaz4.hugops:f3c689e", replicatedRegistry()),
		},
	}
	meta := api.ReleaseMetadata{
		Images: []api.Image{
			{
				URL:      "replicated/www:3",
				Source:   "dockerhub",
				AppSlug:  "awesomeapp",
				ImageKey: "jjzpr9u62gaz2",
			},
			{
				URL:      "quay.io/redacted/chatops:f3c689e",
				Source:   "quayio",
				AppSlug:  "awesomeapp",
				ImageKey: "jjzpr9u62gaz4",
			},
			{
				URL:      "quay.io/redacted/hugops:f3c689e",
				Source:   "quayio",
				ImageKey: "jjzpr9u62gaz4",
			},
		},
	}

	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			r := &URLResolver{Logger: &logger.TestLogger{T: t}}
			url, err := r.ResolvePullURL(test.Asset, meta)
			require.New(t).NoError(err)
			require.New(t).Equal(test.ExpectURL, url)
		})
	}
}
