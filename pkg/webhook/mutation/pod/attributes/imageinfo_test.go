package attributes

import (
	"testing"

	containerattr "github.com/Dynatrace/dynatrace-bootstrapper/cmd/configure/attributes/container"
	"github.com/stretchr/testify/require"
)

func TestCreateImageInfo(t *testing.T) {
	type testCase struct {
		title string
		in    string
		out   containerattr.ImageInfo
	}

	testCases := []testCase{
		{
			title: "empty URI",
			in:    "",
			out:   containerattr.ImageInfo{},
		},
		{
			title: "URI with tag",
			in:    "registry.example.com/repository/image:tag",
			out: containerattr.ImageInfo{
				Registry:    "registry.example.com",
				Repository:  "repository/image",
				Tag:         "tag",
				ImageDigest: "",
			},
		},
		{
			title: "URI with digest",
			in:    "registry.example.com/repository/image@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			out: containerattr.ImageInfo{
				Registry:    "registry.example.com",
				Repository:  "repository/image",
				Tag:         "",
				ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			},
		},
		{
			title: "URI with digest and tag",
			in:    "registry.example.com/repository/image:tag@sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			out: containerattr.ImageInfo{
				Registry:    "registry.example.com",
				Repository:  "repository/image",
				Tag:         "tag",
				ImageDigest: "sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc",
			},
		},
		{
			title: "URI with missing tag",
			in:    "registry.example.com/repository/image",
			out: containerattr.ImageInfo{
				Registry:   "registry.example.com",
				Repository: "repository/image",
			},
		},
		{
			title: "URI with docker.io (special case in certain libraries)",
			in:    "docker.io/php:fpm-stretch",
			out: containerattr.ImageInfo{
				Registry:   "docker.io",
				Repository: "php",
				Tag:        "fpm-stretch",
			},
		},
		{
			title: "URI with missing registry",
			in:    "php:fpm-stretch",
			out: containerattr.ImageInfo{
				Repository: "php",
				Tag:        "fpm-stretch",
			},
		},
	}
	for _, test := range testCases {
		t.Run(test.title, func(t *testing.T) {
			imageInfo := createImageInfo(test.in)

			require.Equal(t, test.out, imageInfo)
		})
	}
}
