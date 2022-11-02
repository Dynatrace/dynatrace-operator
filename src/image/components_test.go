package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var regexTestMap = map[string]Components{
	"nginx":                              {Image: "library/nginx", Version: "latest", Registry: "docker.io"},
	"nginx:1.2.3":                        {Image: "library/nginx", Version: "1.2.3", Registry: "docker.io"},
	"dynatrace/dynatrace-operator":       {Image: "dynatrace/dynatrace-operator", Version: "latest", Registry: "docker.io"},
	"dynatrace/dynatrace-operator:0.9.0": {Image: "dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "docker.io"},
	"dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb": {Image: "dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "docker.io"},
	"quay.io/dynatrace/dynatrace-operator":       {Image: "dynatrace/dynatrace-operator", Version: "latest", Registry: "quay.io"},
	"quay.io/dynatrace/dynatrace-operator:0.9.0": {Image: "dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "quay.io"},
	"quay.io/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb": {Image: "dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "quay.io"},
	"127.0.0.1/dynatrace/dynatrace-operator":       {Image: "dynatrace/dynatrace-operator", Version: "latest", Registry: "127.0.0.1"},
	"127.0.0.1/dynatrace/dynatrace-operator:0.9.0": {Image: "dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "127.0.0.1"},
	"127.0.0.1/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb": {Image: "dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "127.0.0.1"},
	"quay.io:1234/dynatrace/dynatrace-operator":       {Image: "dynatrace/dynatrace-operator", Version: "latest", Registry: "quay.io:1234"},
	"quay.io:1234/dynatrace/dynatrace-operator:0.9.0": {Image: "dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "quay.io:1234"},
	"quay.io:1234/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb": {Image: "dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "quay.io:1234"},
	"127.0.0.1:1234/dynatrace/dynatrace-operator":                                                                                {Image: "dynatrace/dynatrace-operator", Version: "latest", Registry: "127.0.0.1:1234"},
	"127.0.0.1:1234/dynatrace/dynatrace-operator:0.9.0":                                                                          {Image: "dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "127.0.0.1:1234"},
	"127.0.0.1:1234/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb":        {Image: "dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "127.0.0.1:1234"},
	"subdir/dynatrace/dynatrace-operator":                                                                                        {Image: "subdir/dynatrace/dynatrace-operator", Version: "latest", Registry: "docker.io"},
	"subdir/dynatrace/dynatrace-operator:0.9.0":                                                                                  {Image: "subdir/dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "docker.io"},
	"subdir/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb":                {Image: "subdir/dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "docker.io"},
	"quay.io/subdir/dynatrace/dynatrace-operator":                                                                                {Image: "subdir/dynatrace/dynatrace-operator", Version: "latest", Registry: "quay.io"},
	"quay.io/subdir/dynatrace/dynatrace-operator:0.9.0":                                                                          {Image: "subdir/dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "quay.io"},
	"quay.io/subdir/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb":        {Image: "subdir/dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "quay.io"},
	"127.0.0.1/subdir/dynatrace/dynatrace-operator":                                                                              {Image: "subdir/dynatrace/dynatrace-operator", Version: "latest", Registry: "127.0.0.1"},
	"127.0.0.1/subdir/dynatrace/dynatrace-operator:0.9.0":                                                                        {Image: "subdir/dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "127.0.0.1"},
	"127.0.0.1/subdir/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb":      {Image: "subdir/dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "127.0.0.1"},
	"quay.io:1234/subdir/dynatrace/dynatrace-operator":                                                                           {Image: "subdir/dynatrace/dynatrace-operator", Version: "latest", Registry: "quay.io:1234"},
	"quay.io:1234/subdir/dynatrace/dynatrace-operator:0.9.0":                                                                     {Image: "subdir/dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "quay.io:1234"},
	"quay.io:1234/subdir/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb":   {Image: "subdir/dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "quay.io:1234"},
	"127.0.0.1:1234/subdir/dynatrace/dynatrace-operator":                                                                         {Image: "subdir/dynatrace/dynatrace-operator", Version: "latest", Registry: "127.0.0.1:1234"},
	"127.0.0.1:1234/subdir/dynatrace/dynatrace-operator:0.9.0":                                                                   {Image: "subdir/dynatrace/dynatrace-operator", Version: "0.9.0", Registry: "127.0.0.1:1234"},
	"127.0.0.1:1234/subdir/dynatrace/dynatrace-operator@sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb": {Image: "subdir/dynatrace/dynatrace-operator", Version: "sha256:ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb", Registry: "127.0.0.1:1234"},
}

func TestRegex(t *testing.T) {
	for url, expectedComponents := range regexTestMap {
		components, err := ComponentsFromUri(url)

		assert.NoErrorf(t, err, "%s could not be parsed", url)
		assert.Equalf(t, expectedComponents.Registry, components.Registry, "%s has wrong registry", url)
		assert.Equalf(t, expectedComponents.Image, components.Image, "%s has wrong image", url)
		assert.Equalf(t, expectedComponents.Version, components.Version, "%s has wrong version", url)
	}
}
