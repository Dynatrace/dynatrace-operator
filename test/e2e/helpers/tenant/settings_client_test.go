// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package tenant

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

func injectResponse[T any](resp T) func(any) {
	return func(arg any) {
		if target, ok := arg.(*T); ok {
			*target = resp
		}
	}
}

func TestGetEnrichmentRuleObjects(t *testing.T) {
	ctx := t.Context()

	params := map[string]string{
		schemaIDsQueryParam: metadataEnrichmentSchemaID,
		scopesQueryParam:    "KUBERNETES_CLUSTER-123",
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).
			Run(injectResponse(enrichmentRulesObjectsResponse{Items: []EnrichmentRuleObject{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}}})).
			Return(nil).Once()
		apiClient.EXPECT().GET(anyCtx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objects, err := client.GetEnrichmentRuleObjects(ctx, "KUBERNETES_CLUSTER-123")
		require.NoError(t, err)
		assert.Equal(t, []EnrichmentRuleObject{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}}, objects)
	})

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := newSettingsClient(apiClient)
		objects, err := client.GetEnrichmentRuleObjects(ctx, "")
		require.Error(t, err)
		assert.Empty(t, objects)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().GET(anyCtx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objects, err := client.GetEnrichmentRuleObjects(ctx, "KUBERNETES_CLUSTER-123")
		require.Error(t, err)
		assert.Empty(t, objects)
	})
}

func TestCreateEnrichmentRuleObject(t *testing.T) {
	ctx := t.Context()
	const scope = "KUBERNETES_CLUSTER-123"

	rule := metadataenrichment.Rule{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "my-label", Target: "dt.cost.product"}

	matchBody := mock.MatchedBy(func(arg any) bool {
		body, ok := arg.([]postObjectsBody[enrichmentRuleValue])

		return ok &&
			len(body) == 1 &&
			body[0].SchemaID == metadataEnrichmentSchemaID &&
			body[0].Scope == scope &&
			body[0].Value == convertRule(rule)
	})

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).
			Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).
			Return(nil).Once()
		apiClient.EXPECT().POST(ctx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objectIDs, err := client.CreateEnrichmentRuleObject(ctx, scope, rule)
		require.NoError(t, err)
		assert.Equal(t, []string{"obj-123"}, objectIDs)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().POST(ctx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objectIDs, err := client.CreateEnrichmentRuleObject(ctx, scope, rule)
		require.Error(t, err)
		assert.Empty(t, objectIDs)
	})

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := newSettingsClient(apiClient)
		objectIDs, err := client.CreateEnrichmentRuleObject(t.Context(), "", rule)
		require.Error(t, err)
		assert.Empty(t, objectIDs)
	})

	t.Run("multiple rules", func(t *testing.T) {
		rule := metadataenrichment.Rule{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "my-label", Target: "dt.cost.product"}
		rule2 := metadataenrichment.Rule{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "my-label-2", Target: "dt.security_context"}
		matchBody := mock.MatchedBy(func(arg any) bool {
			body, ok := arg.([]postObjectsBody[enrichmentRuleValue])

			return ok &&
				len(body) == 2 &&
				body[0].SchemaID == metadataEnrichmentSchemaID &&
				body[0].Scope == scope &&
				body[0].Value == convertRule(rule) &&
				body[1].SchemaID == metadataEnrichmentSchemaID &&
				body[1].Scope == scope &&
				body[1].Value == convertRule(rule2)
		})

		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).
			Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}})).
			Return(nil).Once()
		apiClient.EXPECT().POST(ctx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)

		objectIDs, err := client.CreateEnrichmentRuleObject(ctx, scope, rule, rule2)
		require.NoError(t, err)
		assert.Equal(t, []string{"obj-1", "obj-2"}, objectIDs)
	})
}

func TestCreateLegacyEnrichmentRuleObject(t *testing.T) {
	ctx := t.Context()
	const scope = "KUBERNETES_CLUSTER-123"

	rule := metadataenrichment.Rule{Type: metadataenrichment.LabelRule, Source: "my-label", Target: "dt.cost.product"}

	matchBody := mock.MatchedBy(func(arg any) bool {
		body, ok := arg.([]postObjectsBody[legacyEnrichmentValue])

		return ok &&
			len(body) == 1 &&
			body[0].SchemaID == legacyMetadataEnrichmentSchemaID &&
			body[0].Scope == scope &&
			len(body[0].Value.Rules) == 1 &&
			body[0].Value.Rules[0] == rule
	})

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).
			Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-456"}})).
			Return(nil).Once()
		apiClient.EXPECT().POST(ctx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objectIDs, err := client.CreateLegacyEnrichmentRuleObject(ctx, scope, rule)
		require.NoError(t, err)
		assert.Equal(t, []string{"obj-456"}, objectIDs)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().POST(ctx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objectIDs, err := client.CreateLegacyEnrichmentRuleObject(ctx, scope, rule)
		require.Error(t, err)
		assert.Empty(t, objectIDs)
	})

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := newSettingsClient(apiClient)
		objectIDs, err := client.CreateLegacyEnrichmentRuleObject(t.Context(), "", rule)
		require.Error(t, err)
		assert.Empty(t, objectIDs)
	})

	t.Run("multiple rules", func(t *testing.T) {
		rule := metadataenrichment.Rule{Type: metadataenrichment.LabelRule, Source: "my-label", Target: "dt.cost.product"}
		rule2 := metadataenrichment.Rule{Type: metadataenrichment.AnnotationRule, Source: "my-label-2", Target: "dt.security_context"}
		matchBody := mock.MatchedBy(func(arg any) bool {
			body, ok := arg.([]postObjectsBody[legacyEnrichmentValue])

			return ok &&
				len(body) == 1 &&
				body[0].SchemaID == legacyMetadataEnrichmentSchemaID &&
				body[0].Scope == scope &&
				len(body[0].Value.Rules) == 2 &&
				body[0].Value.Rules[0] == rule &&
				body[0].Value.Rules[1] == rule2
		})

		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(map[string]string{validateOnlyQueryParam: "false"}).Return(request).Once()
		request.EXPECT().WithJSONBody(matchBody).Return(request).Once()
		request.EXPECT().Execute(new([]postObjectsResponse)).
			Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-456"}})).
			Return(nil).Once()
		apiClient.EXPECT().POST(ctx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objectIDs, err := client.CreateLegacyEnrichmentRuleObject(ctx, scope, rule, rule2)
		require.NoError(t, err)
		assert.Equal(t, []string{"obj-456"}, objectIDs)
	})
}

func TestGetLegacyEnrichmentRuleObjects(t *testing.T) {
	ctx := t.Context()

	params := map[string]string{
		schemaIDsQueryParam: legacyMetadataEnrichmentSchemaID,
		scopesQueryParam:    "KUBERNETES_CLUSTER-123",
	}

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).
			Run(injectResponse(enrichmentRulesObjectsResponse{Items: []EnrichmentRuleObject{{ObjectID: "obj-1"}}})).
			Return(nil).Once()
		apiClient.EXPECT().GET(anyCtx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objects, err := client.GetLegacyEnrichmentRuleObjects(ctx, "KUBERNETES_CLUSTER-123")
		require.NoError(t, err)
		assert.Equal(t, []EnrichmentRuleObject{{ObjectID: "obj-1"}}, objects)
	})

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := newSettingsClient(apiClient)
		objects, err := client.GetLegacyEnrichmentRuleObjects(ctx, "")
		require.Error(t, err)
		assert.Empty(t, objects)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(params).Return(request).Once()
		request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).Return(errors.New("api error")).Once()
		apiClient.EXPECT().GET(anyCtx, objectsPath).Return(request).Once()

		client := newSettingsClient(apiClient)
		objects, err := client.GetLegacyEnrichmentRuleObjects(ctx, "KUBERNETES_CLUSTER-123")
		require.Error(t, err)
		assert.Empty(t, objects)
	})
}

func TestDeleteSettings(t *testing.T) {
	ctx := t.Context()
	objectID := "settings-object-123"

	t.Run("success", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().Execute(nil).Return(nil).Once()
		apiClient.EXPECT().DELETE(ctx, "/v2/settings/objects/"+objectID).Return(request).Once()

		client := newSettingsClient(apiClient)
		err := client.DeleteSettings(ctx, objectID)
		require.NoError(t, err)
	})

	t.Run("error from API", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().Execute(nil).Return(errors.New("api error")).Once()
		apiClient.EXPECT().DELETE(ctx, "/v2/settings/objects/"+objectID).Return(request).Once()

		client := newSettingsClient(apiClient)
		err := client.DeleteSettings(ctx, objectID)
		require.Error(t, err)
		assert.ErrorIs(t, err, errDeleteSettings)
	})

	t.Run("empty objectID", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := newSettingsClient(apiClient)
		err := client.DeleteSettings(ctx, "")
		require.Error(t, err)
		assert.ErrorIs(t, err, errNoSettingsIDProvided)
	})
}
