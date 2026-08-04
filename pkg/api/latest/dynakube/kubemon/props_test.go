// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package kubemon_test

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/kubemon"
	"github.com/stretchr/testify/assert"
)

func TestGetDefaultImage(t *testing.T) {
	const (
		apiURLHost  = "tenant.live.dynatrace.com"
		imagePrefix = apiURLHost + kubemon.TenantRegistrySubPath + ":"
	)

	newKubeMon := func(urlHost string) *kubemon.KubeMon {
		km := &kubemon.KubeMon{Spec: &kubemon.Spec{}, Status: &kubemon.Status{}}
		km.SetAPIURLHost(urlHost)

		return km
	}

	t.Run("4-part version is truncated to 3 parts", func(t *testing.T) {
		assert.Equal(t, imagePrefix+"1.2.3-raw", newKubeMon(apiURLHost).GetDefaultImage("1.2.3.4"))
	})

	t.Run("3-part version is kept as-is", func(t *testing.T) {
		assert.Equal(t, imagePrefix+"1.2.3-raw", newKubeMon(apiURLHost).GetDefaultImage("1.2.3"))
	})

	t.Run("version already containing raw is not doubled", func(t *testing.T) {
		assert.Equal(t, imagePrefix+"1.2.3-raw", newKubeMon(apiURLHost).GetDefaultImage("1.2.3-raw"))
	})

	t.Run("empty api url returns empty string", func(t *testing.T) {
		assert.Empty(t, newKubeMon("").GetDefaultImage("1.2.3"))
	})

	t.Run("full image path contains host, subpath, and tag", func(t *testing.T) {
		image := newKubeMon(apiURLHost).GetDefaultImage("1.2.3")
		assert.Contains(t, image, apiURLHost)
		assert.Contains(t, image, kubemon.TenantRegistrySubPath)
		assert.Contains(t, image, "1.2.3-raw")
	})
}
