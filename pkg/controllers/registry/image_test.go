// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/image"
	imagemock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/image"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeResolver struct{ spec, status, override string }

func (r *fakeResolver) GetImage() string               { return r.spec }
func (r *fakeResolver) SetImage(img string)            { r.status = img }
func (r *fakeResolver) PublicRegistryOverride() string { return r.override }

func TestResolveImage(t *testing.T) {
	const testURI = "registry.example.com/dynatrace/eec:1.2.3"
	const testOverride = "custom.registry.example.com"

	t.Run("set from spec", func(t *testing.T) {
		r := &fakeResolver{spec: testURI}
		err := ResolveImage(t.Context(), r, nil, image.EEC)
		require.NoError(t, err)
		assert.Equal(t, testURI, r.status)
	})

	t.Run("returns URI from public registry", func(t *testing.T) {
		r := &fakeResolver{}
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.EEC, "").Return(&image.Info{URI: testURI}, nil)

		err := ResolveImage(t.Context(), r, mockClient, image.EEC)
		require.NoError(t, err)
		assert.Equal(t, testURI, r.status)
	})

	t.Run("passes registryOverride to client", func(t *testing.T) {
		r := &fakeResolver{override: testOverride}
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.DBExecutor, testOverride).Return(&image.Info{URI: testURI}, nil)

		err := ResolveImage(t.Context(), r, mockClient, image.DBExecutor)
		require.NoError(t, err)
		assert.Equal(t, testURI, r.status)
	})

	t.Run("propagates error from client", func(t *testing.T) {
		r := &fakeResolver{}
		mockClient := imagemock.NewClient(t)
		boom := errors.New("connection refused")
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.LogModule, "").Return(nil, boom)

		err := ResolveImage(t.Context(), r, mockClient, image.LogModule)
		require.ErrorIs(t, err, boom)
		assert.Empty(t, r.status)
	})

	t.Run("no fallback on API error", func(t *testing.T) {
		r := &fakeResolver{}
		mockClient := imagemock.NewClient(t)
		boom := &core.HTTPError{StatusCode: 404}
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.LogModule, "").Return(nil, boom)

		err := ResolveImage(t.Context(), r, mockClient, image.LogModule, image.EEC)
		require.ErrorIs(t, err, boom)
	})

	t.Run("breaks fallback on API error", func(t *testing.T) {
		r := &fakeResolver{}
		mockClient := imagemock.NewClient(t)
		boom := &core.HTTPError{StatusCode: 404}
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.LogModule, "").Return(nil, errors.New("image not found"))
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.EEC, "").Return(nil, boom)

		err := ResolveImage(t.Context(), r, mockClient, image.LogModule, image.EEC, image.ActiveGate)
		require.ErrorIs(t, err, boom)
	})

	t.Run("returns last fallback error", func(t *testing.T) {
		r := &fakeResolver{}
		mockClient := imagemock.NewClient(t)
		boom := errors.New("boom")
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.LogModule, "").Return(nil, errors.New("image not found"))
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.EEC, "").Return(nil, errors.New("this neither"))
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.ActiveGate, "").Return(nil, boom)

		err := ResolveImage(t.Context(), r, mockClient, image.LogModule, image.EEC, image.ActiveGate)
		require.ErrorIs(t, err, boom)
	})

	t.Run("use fallback", func(t *testing.T) {
		r := &fakeResolver{}
		mockClient := imagemock.NewClient(t)
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.LogModule, "").Return(nil, errors.New("image not found"))
		mockClient.EXPECT().GetComponentLatestInfo(t.Context(), image.EEC, "").Return(&image.Info{URI: testURI}, nil)

		err := ResolveImage(t.Context(), r, mockClient, image.LogModule, image.EEC, image.ActiveGate)
		require.NoError(t, err)
		assert.Equal(t, testURI, r.status)
	})
}
