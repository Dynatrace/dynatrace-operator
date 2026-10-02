// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package settings

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	coremock "github.com/Dynatrace/dynatrace-operator/test/mocks/pkg/clients/dynatrace/core"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var anyCtx = mock.MatchedBy(func(context.Context) bool { return true })

// rulesPath and rulesParams let TestGetRulesSetting run its whole table against both generations.
func rulesPath(generation Generation) string {
	if generation == Gen3 {
		return gen3EffectiveValuesPath
	}

	return gen2EffectiveValuesPath
}

func rulesParams(generation Generation, schemaID, scope string) map[string]string {
	if generation == Gen3 {
		return map[string]string{gen3SchemaIDParam: schemaID, gen3ScopeParam: scope}
	}

	return map[string]string{gen2ValidateOnlyParam: "true", gen2SchemaIDsParam: schemaID, gen2ScopeParam: scope}
}

// rulesScope mirrors objectAPI.scope for GetRules: gen2 uses the Monitored Entity ID directly, falling
// back to the global environment scope when it is not yet known. Gen3 reuses the Monitored Entity ID
// too once known; only when it is still empty does gen3 fall back to deriving a scope from
// EntityID (via the Smartscape cluster-UID lookup) instead of the environment scope.
func rulesScope(generation Generation, kubeSystemUUID, meid string) string {
	if meid != "" {
		return meid
	}

	if generation == Gen3 {
		return k8sClusterUIDScope(kubeSystemUUID)
	}

	return globalScope
}

func TestGetRulesSetting(t *testing.T) {
	for _, generation := range []Generation{Gen2, Gen3} {
		t.Run(map[Generation]string{Gen2: "gen2", Gen3: "gen3"}[generation], func(t *testing.T) {
			testGetRulesSetting(t, generation)
		})
	}
}

func testGetRulesSetting(t *testing.T, generation Generation) {
	t.Helper()

	ctx := t.Context()
	path := rulesPath(generation)

	oldParams := rulesParams(generation, legacyMetadataEnrichmentSchemaID, rulesScope(generation, "kube-system-uuid", "ENVIRONMENT_ID"))
	newParams := rulesParams(generation, metadataEnrichmentSchemaID, rulesScope(generation, "kube-system-uuid", "ENVIRONMENT_ID"))

	expectRules := []metadataenrichment.Rule{
		{Type: metadataenrichment.LabelRule, Source: "source-1", Target: "target-1"},
		{Type: metadataenrichment.AnnotationRule, Source: "source-2", Target: "target-2"},
	}

	oldResponse := getRulesResponse{
		Items: []ruleItem{
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "source-1", Target: "target-1"},
						{Type: metadataenrichment.AnnotationRule, Source: "source-2", Target: "target-2"},
					},
				},
			},
		},
	}

	newResponse := getRulesResponse{
		Items: []ruleItem{
			{Value: ruleItemValue{Type: metadataenrichment.K8sNamespaceLabelRule, ValueSource: "source-1", Target: "target-1"}},
			{Value: ruleItemValue{Type: metadataenrichment.K8sNamespaceAnnotationRule, ValueSource: "source-2", Target: "target-2"}},
			{Value: ruleItemValue{Type: "FOO", ValueSource: "source-3", Target: "target-3"}},
			{Value: ruleItemValue{Type: metadataenrichment.CustomRule, ValueSource: "source-4", Target: "target-4", Condition: "true"}},
		},
	}

	setFlag := func(resp getRulesResponse) getRulesResponse {
		items := make([]ruleItem, len(resp.Items))
		copy(items, resp.Items)

		for i := range items {
			items[i].Value.UseIngestEnrichmentConfigSchema = true
		}

		return getRulesResponse{Items: items}
	}

	t.Run("get rules", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(oldParams).Return(request).Once()
		request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(oldResponse)).Return(nil).Once()
		apiClient.EXPECT().GET(anyCtx, path).Return(request).Once()

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid", EntityScope: "ENVIRONMENT_ID"})
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("no kubesystem-uuid -> error", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		settingsClient := NewClient(apiClient, generation)
		rules, err := settingsClient.GetRules(ctx, K8sClusterRegistration{EntityScope: "test-entityID"})
		require.ErrorIs(t, err, errMissingKubeSystemUUID)
		assert.Empty(t, rules)
	})

	t.Run("non 404 error", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		request.EXPECT().WithQueryParams(oldParams).Return(request).Once()
		httpErr := &core.HTTPError{StatusCode: 503}
		request.EXPECT().Execute(new(getRulesResponse)).Return(httpErr).Once()
		apiClient.EXPECT().GET(anyCtx, path).Return(request).Once()

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid", EntityScope: "ENVIRONMENT_ID"})
		require.ErrorIs(t, err, httpErr)
		assert.Empty(t, rules)
	})

	t.Run("no monitored-entities, use environment scope -> return not-empty, no error", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		// Gen2 should use globalScope ("environment") for scope; gen3 always has a scope (the MEID
		// is empty here, so it derives one from EntityID instead) so it never falls back to the
		// environment scope.
		request.EXPECT().WithQueryParams(rulesParams(generation, legacyMetadataEnrichmentSchemaID, rulesScope(generation, "kube-system-uuid", ""))).Return(request).Once()
		request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(oldResponse)).Return(nil).Once()
		apiClient.EXPECT().GET(anyCtx, path).Return(request).Once()

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid"})
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("new schema enabled explicitly", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		expectRules := []metadataenrichment.Rule{
			{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "source-1", Target: "target-1"},
			{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "source-2", Target: "target-2"},
		}

		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(setFlag(oldResponse))).Return(nil).Once(),
			// Switch to new schema
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(newResponse)).Return(nil).Once(),
		)

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid", EntityScope: "ENVIRONMENT_ID"})
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("use new schema with old empty rules", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		expectRules := []metadataenrichment.Rule{
			{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "source-1", Target: "target-1"},
			{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "source-2", Target: "target-2"},
		}

		// No rules defined
		oldResponse := getRulesResponse{Items: []ruleItem{{}}}

		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(setFlag(oldResponse))).Return(nil).Once(),
			// Switch to new schema
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(newResponse)).Return(nil).Once(),
		)

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid", EntityScope: "ENVIRONMENT_ID"})
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("new enrichment settings schema not available", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		httpErr := &core.HTTPError{StatusCode: 404}

		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(setFlag(oldResponse))).Return(nil).Once(),
			// Switch to new schema
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(httpErr).Once(),
		)

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid", EntityScope: "ENVIRONMENT_ID"})
		require.Error(t, err)
		assert.Empty(t, rules)
	})

	t.Run("use enrichment settings schema fallback", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		expectRules := []metadataenrichment.Rule{
			{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "source-1", Target: "target-1"},
			{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "source-2", Target: "target-2"},
		}

		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(&core.HTTPError{StatusCode: 404}).Once(),
			// Fallback after 404 with old schema
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Run(injectResponse(newResponse)).Return(nil).Once(),
		)

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid", EntityScope: "ENVIRONMENT_ID"})
		require.NoError(t, err)
		assert.Equal(t, expectRules, rules)
	})

	t.Run("neither enrichment settings schema available", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		request := coremock.NewRequest(t)
		expectCallOrder(
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(oldParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(&core.HTTPError{StatusCode: 404}).Once(),
			apiClient.EXPECT().GET(anyCtx, path).Return(request).Once(),
			request.EXPECT().WithQueryParams(newParams).Return(request).Once(),
			request.EXPECT().Execute(new(getRulesResponse)).Return(&core.HTTPError{StatusCode: 404}).Once(),
		)

		client := NewClient(apiClient, generation)
		rules, err := client.GetRules(ctx, K8sClusterRegistration{EntityID: "kube-system-uuid", EntityScope: "ENVIRONMENT_ID"})
		require.NoError(t, err)
		assert.Empty(t, rules)
	})
}

// Set up mocked calls to verify they are executed in the input order.
func expectCallOrder(calls ...*mock.Call) {
	if len(calls) < 2 {
		return
	}

	prev := calls[0]

	for _, call := range calls[1:] {
		call.NotBefore(prev)
		prev = call
	}
}

func TestGetEnrichmentRuleObjects(t *testing.T) {
	ctx := t.Context()

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		objects, err := client.GetEnrichmentRuleObjects(ctx, K8sClusterRegistration{})
		require.Error(t, err)
		assert.Empty(t, objects)
	})

	t.Run("gen2", func(t *testing.T) {
		params := map[string]string{
			gen2SchemaIDsParam: metadataEnrichmentSchemaID,
			gen2ScopesParam:    "KUBERNETES_CLUSTER-123",
		}
		registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "KUBERNETES_CLUSTER-123"}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).
				Run(injectResponse(enrichmentRulesObjectsResponse{Items: []EnrichmentRuleObject{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}}})).
				Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objects, err := client.GetEnrichmentRuleObjects(ctx, registration)
			require.NoError(t, err)
			assert.Equal(t, []EnrichmentRuleObject{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}}, objects)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().GET(anyCtx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objects, err := client.GetEnrichmentRuleObjects(ctx, registration)
			require.Error(t, err)
			assert.Empty(t, objects)
		})
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		params := map[string]string{
			gen3SchemaIDParam: metadataEnrichmentSchemaID,
			gen3ScopeParam:    "KUBERNETES_CLUSTER-123",
		}
		registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "KUBERNETES_CLUSTER-123"}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).
				Run(injectResponse(enrichmentRulesObjectsResponse{Items: []EnrichmentRuleObject{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}}})).
				Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objects, err := client.GetEnrichmentRuleObjects(ctx, registration)
			require.NoError(t, err)
			assert.Equal(t, []EnrichmentRuleObject{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}}, objects)
		})
	})
}

func TestCreateEnrichmentRuleObject(t *testing.T) {
	ctx := t.Context()
	registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "KUBERNETES_CLUSTER-123"}
	const gen3Scope = "KUBERNETES_CLUSTER-123"

	rule := metadataenrichment.Rule{Type: metadataenrichment.K8sNamespaceLabelRule, Source: "my-label", Target: "dt.cost.product"}

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		objectIDs, err := client.CreateEnrichmentRuleObject(t.Context(), K8sClusterRegistration{}, rule)
		require.Error(t, err)
		assert.Empty(t, objectIDs)
	})

	t.Run("gen2 uses the Monitored Entity ID as scope", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(metadataEnrichmentSchemaID, "", "KUBERNETES_CLUSTER-123", convertRule(rule))).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).
				Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-123"}})).
				Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectIDs, err := client.CreateEnrichmentRuleObject(ctx, registration, rule)
			require.NoError(t, err)
			assert.Equal(t, []string{"obj-123"}, objectIDs)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(metadataEnrichmentSchemaID, "", "KUBERNETES_CLUSTER-123", convertRule(rule))).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectIDs, err := client.CreateEnrichmentRuleObject(ctx, registration, rule)
			require.Error(t, err)
			assert.Empty(t, objectIDs)
		})

		t.Run("multiple rules batched into one request", func(t *testing.T) {
			rule2 := metadataenrichment.Rule{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "my-label-2", Target: "dt.security_context"}

			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(metadataEnrichmentSchemaID, "", "KUBERNETES_CLUSTER-123", convertRule(rule), convertRule(rule2))).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).
				Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-1"}, {ObjectID: "obj-2"}})).
				Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)

			objectIDs, err := client.CreateEnrichmentRuleObject(ctx, registration, rule, rule2)
			require.NoError(t, err)
			assert.Equal(t, []string{"obj-1", "obj-2"}, objectIDs)
		})
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(metadataEnrichmentSchemaID, "", gen3Scope, convertRule(rule))).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).
				Run(injectResponse(postObjectsResponse{ObjectID: "obj-123"})).
				Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectIDs, err := client.CreateEnrichmentRuleObject(ctx, registration, rule)
			require.NoError(t, err)
			assert.Equal(t, []string{"obj-123"}, objectIDs)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(metadataEnrichmentSchemaID, "", gen3Scope, convertRule(rule))).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectIDs, err := client.CreateEnrichmentRuleObject(ctx, registration, rule)
			require.Error(t, err)
			assert.Empty(t, objectIDs)
		})

		t.Run("multiple rules issue one request per rule", func(t *testing.T) {
			rule2 := metadataenrichment.Rule{Type: metadataenrichment.K8sNamespaceAnnotationRule, Source: "my-label-2", Target: "dt.security_context"}

			apiClient := coremock.NewClient(t)

			request1 := coremock.NewRequest(t)
			request1.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request1).Once()
			request1.EXPECT().WithJSONBody(matchGen3Body(metadataEnrichmentSchemaID, "", gen3Scope, convertRule(rule))).Return(request1).Once()
			request1.EXPECT().Execute(new(postObjectsResponse)).
				Run(injectResponse(postObjectsResponse{ObjectID: "obj-1"})).
				Return(nil).Once()

			request2 := coremock.NewRequest(t)
			request2.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request2).Once()
			request2.EXPECT().WithJSONBody(matchGen3Body(metadataEnrichmentSchemaID, "", gen3Scope, convertRule(rule2))).Return(request2).Once()
			request2.EXPECT().Execute(new(postObjectsResponse)).
				Run(injectResponse(postObjectsResponse{ObjectID: "obj-2"})).
				Return(nil).Once()

			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request1).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request2).Once()

			client := NewClient(apiClient, Gen3)

			objectIDs, err := client.CreateEnrichmentRuleObject(ctx, registration, rule, rule2)
			require.NoError(t, err)
			assert.Equal(t, []string{"obj-1", "obj-2"}, objectIDs)
		})
	})
}

func TestCreateLegacyEnrichmentRuleObject(t *testing.T) {
	ctx := t.Context()

	rule := metadataenrichment.Rule{Type: metadataenrichment.LabelRule, Source: "my-label", Target: "dt.cost.product"}
	rule2 := metadataenrichment.Rule{Type: metadataenrichment.AnnotationRule, Source: "my-label-2", Target: "dt.security_context"}
	value := legacyEnrichmentValue{Rules: []metadataenrichment.Rule{rule, rule2}}

	registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "KUBERNETES_CLUSTER-123"}
	const gen3Scope = "KUBERNETES_CLUSTER-123"

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		objectIDs, err := client.CreateLegacyEnrichmentRuleObject(t.Context(), K8sClusterRegistration{}, rule)
		require.Error(t, err)
		assert.Empty(t, objectIDs)
	})

	t.Run("gen2 uses the Monitored Entity ID as scope", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(legacyMetadataEnrichmentSchemaID, "", "KUBERNETES_CLUSTER-123", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).
				Run(injectResponse([]postObjectsResponse{{ObjectID: "obj-456"}})).
				Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectIDs, err := client.CreateLegacyEnrichmentRuleObject(ctx, registration, rule, rule2)
			require.NoError(t, err)
			assert.Equal(t, []string{"obj-456"}, objectIDs)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen2ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen2Body(legacyMetadataEnrichmentSchemaID, "", "KUBERNETES_CLUSTER-123", value)).Return(request).Once()
			request.EXPECT().Execute(new([]postObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().POST(ctx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objectIDs, err := client.CreateLegacyEnrichmentRuleObject(ctx, registration, rule, rule2)
			require.Error(t, err)
			assert.Empty(t, objectIDs)
		})
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(map[string]string{gen3ValidateOnlyParam: "false"}).Return(request).Once()
			request.EXPECT().WithJSONBody(matchGen3Body(legacyMetadataEnrichmentSchemaID, "", gen3Scope, value)).Return(request).Once()
			request.EXPECT().Execute(new(postObjectsResponse)).
				Run(injectResponse(postObjectsResponse{ObjectID: "obj-456"})).
				Return(nil).Once()
			apiClient.EXPECT().POST(ctx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objectIDs, err := client.CreateLegacyEnrichmentRuleObject(ctx, registration, rule, rule2)
			require.NoError(t, err)
			assert.Equal(t, []string{"obj-456"}, objectIDs)
		})
	})
}

func TestGetLegacyEnrichmentRuleObjects(t *testing.T) {
	ctx := t.Context()

	registration := K8sClusterRegistration{EntityID: "uuid-1", EntityScope: "KUBERNETES_CLUSTER-123"}

	t.Run("empty scope", func(t *testing.T) {
		apiClient := coremock.NewClient(t)
		client := NewClient(apiClient, Gen2)
		objects, err := client.GetLegacyEnrichmentRuleObjects(ctx, K8sClusterRegistration{})
		require.Error(t, err)
		assert.Empty(t, objects)
	})

	t.Run("gen2", func(t *testing.T) {
		params := map[string]string{
			gen2SchemaIDsParam: legacyMetadataEnrichmentSchemaID,
			gen2ScopesParam:    "KUBERNETES_CLUSTER-123",
		}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).
				Run(injectResponse(enrichmentRulesObjectsResponse{Items: []EnrichmentRuleObject{{ObjectID: "obj-1"}}})).
				Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objects, err := client.GetLegacyEnrichmentRuleObjects(ctx, registration)
			require.NoError(t, err)
			assert.Equal(t, []EnrichmentRuleObject{{ObjectID: "obj-1"}}, objects)
		})

		t.Run("error from API", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).Return(errors.New("api error")).Once()
			apiClient.EXPECT().GET(anyCtx, gen2ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen2)
			objects, err := client.GetLegacyEnrichmentRuleObjects(ctx, registration)
			require.Error(t, err)
			assert.Empty(t, objects)
		})
	})

	t.Run("gen3 reuses EntityScope once known", func(t *testing.T) {
		params := map[string]string{
			gen3SchemaIDParam: legacyMetadataEnrichmentSchemaID,
			gen3ScopeParam:    "KUBERNETES_CLUSTER-123",
		}

		t.Run("success", func(t *testing.T) {
			apiClient := coremock.NewClient(t)
			request := coremock.NewRequest(t)
			request.EXPECT().WithQueryParams(params).Return(request).Once()
			request.EXPECT().Execute(new(enrichmentRulesObjectsResponse)).
				Run(injectResponse(enrichmentRulesObjectsResponse{Items: []EnrichmentRuleObject{{ObjectID: "obj-1"}}})).
				Return(nil).Once()
			apiClient.EXPECT().GET(anyCtx, gen3ObjectsPath).Return(request).Once()

			client := NewClient(apiClient, Gen3)
			objects, err := client.GetLegacyEnrichmentRuleObjects(ctx, registration)
			require.NoError(t, err)
			assert.Equal(t, []EnrichmentRuleObject{{ObjectID: "obj-1"}}, objects)
		})
	})
}

// This is just a sanity check that the models match what's returned by the API.
func Test_enrichmentSchemaModel(t *testing.T) {
	const rawDataOld = `{"items":[{"origin":"environment","value":{"rules":[{"type":"LABEL","source":"test-cost","target":"dt.cost.product"},{"type":"ANNOTATION","source":"my.test.annotation/value","target":"dt.security_context"}]}}],"totalCount":1,"pageSize":100}`
	const rawDataOldFlag = `{"items":[{"origin":"environment","value":{"rules":[{"type":"LABEL","source":"test-cost","target":"dt.cost.product"},{"type":"ANNOTATION","source":"my.test.annotation/value","target":"dt.security_context"}],"useIngestEnrichmentConfigSchema":true}}],"totalCount":1,"pageSize":100}`
	const rawDataNew = `{"items":[{"origin":"environment","value":{"type":"K8S_NAMESPACE_LABEL","valueSource":"test-label","target":"dt.cost.product"}},{"origin":"environment","value":{"type":"K8S_NAMESPACE_ANNOTATION","valueSource":"my.test.annotation/value","target":"dt.security_context"}}],"totalCount":2,"pageSize":100}`

	expectOld := getRulesResponse{
		Items: []ruleItem{
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "test-cost", Target: "dt.cost.product"},
						{Type: metadataenrichment.AnnotationRule, Source: "my.test.annotation/value", Target: "dt.security_context"},
					},
				},
			},
		},
	}

	expectOldFlag := getRulesResponse{
		Items: []ruleItem{
			{
				Value: ruleItemValue{
					Rules: []metadataenrichment.Rule{
						{Type: metadataenrichment.LabelRule, Source: "test-cost", Target: "dt.cost.product"},
						{Type: metadataenrichment.AnnotationRule, Source: "my.test.annotation/value", Target: "dt.security_context"},
					},
					UseIngestEnrichmentConfigSchema: true,
				},
			},
		},
	}

	expectNew := getRulesResponse{
		Items: []ruleItem{
			{Value: ruleItemValue{Type: metadataenrichment.K8sNamespaceLabelRule, ValueSource: "test-label", Target: "dt.cost.product"}},
			{Value: ruleItemValue{Type: metadataenrichment.K8sNamespaceAnnotationRule, ValueSource: "my.test.annotation/value", Target: "dt.security_context"}},
		},
	}

	tests := []struct {
		name   string
		input  string
		expect getRulesResponse
	}{
		{"old", rawDataOld, expectOld},
		{"old with flag", rawDataOldFlag, expectOldFlag},
		{"new", rawDataNew, expectNew},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var resp getRulesResponse
			require.NoError(t, json.Unmarshal([]byte(test.input), &resp))
			assert.Equal(t, test.expect, resp)
		})
	}
}

type capturingLogSink struct {
	values []any
}

func (c *capturingLogSink) Enabled(level int) bool                            { return true }
func (c *capturingLogSink) Error(err error, msg string, keysAndValues ...any) {}
func (c *capturingLogSink) Init(info logr.RuntimeInfo)                        {}
func (c *capturingLogSink) WithName(name string) logr.LogSink                 { return c }
func (c *capturingLogSink) WithValues(keysAndValues ...any) logr.LogSink      { return c }
func (c *capturingLogSink) Info(level int, msg string, keysAndValues ...any) {
	c.values = append(c.values, keysAndValues...)
}

func TestLogDroppedRules(t *testing.T) {
	tests := []struct {
		name   string
		resp   getRulesResponse
		expect []any
	}{
		{"no rules", getRulesResponse{}, nil},
		{
			"legacy rules",
			getRulesResponse{
				Items: []ruleItem{
					{Value: ruleItemValue{Rules: []metadataenrichment.Rule{
						{Type: "a", Source: "a"},
						{Type: "b", Source: "b"},
						{Type: "c", Source: "c"},
					}}},
				},
			},
			nil,
		},
		{
			"no dropped rules",
			getRulesResponse{
				Items: []ruleItem{
					{Value: ruleItemValue{Type: "K8S_NAMESPACE_LABEL", Target: "a", ValueSource: "a"}},
					{Value: ruleItemValue{Type: "K8S_NAMESPACE_LABEL", Target: "b", ValueSource: "b"}},
				},
			},
			nil,
		},
		{
			"log dropped rules",
			getRulesResponse{
				Items: []ruleItem{
					{Value: ruleItemValue{Type: "K8S_NAMESPACE_LABEL", Target: "a", ValueSource: "a"}},
					{Value: ruleItemValue{Type: "K8S_NAMESPACE_LABEL", Target: "b", ValueSource: "b", Condition: "b"}},
					{Value: ruleItemValue{Type: "FOO", Target: "c", ValueSource: "c"}},
					{Value: ruleItemValue{Type: "K8S_NAMESPACE_LABEL", ValueSource: "d"}},
				},
			},
			[]any{
				"rules",
				[]ingestEnrichmentConfig{
					{Type: "K8S_NAMESPACE_LABEL", Target: "b", ValueSource: "b", Condition: "b"},
					{Type: "FOO", Target: "c", ValueSource: "c"},
					{Type: "K8S_NAMESPACE_LABEL", ValueSource: "d"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logSink capturingLogSink
			ctx := logr.NewContext(t.Context(), logr.New(&logSink))
			_ = getRulesFromResponse(ctx, tt.resp)
			assert.Equal(t, tt.expect, logSink.values)
		})
	}
}
