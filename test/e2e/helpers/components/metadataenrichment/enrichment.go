// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package metadataenrichment

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/system"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func getKubeSystemUUID(ctx context.Context, t *testing.T, envConfig *envconf.Config) string {
	t.Helper()

	uid, err := system.GetUID(ctx, envConfig.Client().Resources().GetControllerRuntimeClient())
	require.NoError(t, err)

	return string(uid)
}

// EnsureKubernetesClusterMEID creates the builtin:cloud.kubernetes setting if not present,
// triggering the creation of the Kubernetes Cluster Monitored Entity on the tenant.
func EnsureKubernetesClusterMEID(secretConfig tenant.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		settingsClient, err := tenant.BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		kubeSystemUUID := getKubeSystemUUID(ctx, t, envConfig)
		t.Logf("kube-system UUID: %s", kubeSystemUUID)

		existingME, err := settingsClient.GetK8sClusterME(ctx, kubeSystemUUID)
		require.NoError(t, err)

		if existingME.ID == "" {
			_, err = settingsClient.CreateOrUpdateKubernetesSetting(ctx, "e2e-enrichment-test", kubeSystemUUID, "")
			require.NoError(t, err, "Could not create Kubernetes connection setting")
		}

		var meid string

		err = retry.OnError(retry.DefaultRetry, func(err error) bool { return err != nil }, func() error {
			me, retryErr := settingsClient.GetK8sClusterME(ctx, kubeSystemUUID)
			if retryErr != nil {
				return retryErr
			}

			if me.ID == "" {
				return errors.New("kubernetes cluster MEID not yet available")
			}

			meid = me.ID

			return nil
		})
		require.NoError(t, err, "Kubernetes Cluster MEID did not become available after settings creation")
		t.Logf("Kubernetes Cluster MEID: %s", meid)

		return ctx
	}
}

type rulesStateKey struct{}

// rulesState tracks what the test changed on the tenant, so that it can be reverted without touching anything else.
type rulesState struct {
	scope string
	// legacyBackup holds the value of the legacy schema object that existed before the test, nil if there was none.
	legacyBackup json.RawMessage
	// createdNewSchema holds the IDs of the objects the test created with the new schema.
	createdNewSchema []string
}

func getRulesState(ctx context.Context, t *testing.T) *rulesState {
	t.Helper()

	state, ok := ctx.Value(rulesStateKey{}).(*rulesState)
	require.True(t, ok, "BackupEnrichmentRulesOnTenant must run before changing enrichment rules")

	return state
}

// getLegacyRuleObject returns the legacy schema object of the scope, nil if there is none.
// The schema allows at most one object per scope.
func getLegacyRuleObject(ctx context.Context, t *testing.T, settingsClient *tenant.SettingsClient, scope string) *tenant.RawObject {
	t.Helper()

	objects, err := settingsClient.GetRawObjects(ctx, tenant.LegacyMetadataEnrichmentSchemaID, scope)
	if core.IsNotFound(err) {
		// the schema is not available on the tenant, so there is nothing to find
		return nil
	}

	require.NoError(t, err, "Could not get legacy enrichment rule object")
	require.LessOrEqual(t, len(objects), 1, "The legacy enrichment schema allows a single object per scope")

	if len(objects) == 0 {
		return nil
	}

	return &objects[0]
}

func deleteObject(ctx context.Context, t *testing.T, settingsClient *tenant.SettingsClient, objectID string) {
	t.Helper()

	if err := settingsClient.DeleteSettings(ctx, objectID); err != nil {
		t.Errorf("Failed to delete enrichment rule %s: %v", objectID, err)
	} else {
		t.Logf("Deleted enrichment rule: %s", objectID)
	}
}

// BackupEnrichmentRulesOnTenant prepares the tenant for creating enrichment rules without losing the existing ones.
// The legacy schema (builtin:kubernetes.generic.metadata.enrichment) only allows a single object per scope, which can't be extended,
// so the existing object is stored and removed. RestoreEnrichmentRulesOnTenant brings it back.
// The new schema (builtin:ingest.enrichment.config) supports multiple objects, so the existing rules are left alone.
func BackupEnrichmentRulesOnTenant(secretConfig tenant.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		settingsClient, err := tenant.BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		k8sClusterME, err := settingsClient.GetK8sClusterME(ctx, getKubeSystemUUID(ctx, t, envConfig))
		require.NoError(t, err, "Could not get K8s cluster MEID")
		require.NotEmpty(t, k8sClusterME.ID, "Kubernetes Cluster MEID must exist before backing up enrichment rules")

		existing := getLegacyRuleObject(ctx, t, settingsClient, k8sClusterME.ID)

		state := &rulesState{scope: k8sClusterME.ID}

		if existing != nil {
			state.legacyBackup = existing.Value
			// Logged so that the rules can be recovered manually if the restore fails.
			t.Logf("Backing up legacy enrichment rule object (scope: %s): %s", state.scope, string(existing.Value))
		}

		// Register the state before deleting anything, so the restore always knows what to put back.
		ctx = context.WithValue(ctx, rulesStateKey{}, state)

		if existing != nil {
			deleteObject(ctx, t, settingsClient, existing.ObjectID)
		}

		return ctx
	}
}

// CreateEnrichmentRuleOnTenant creates the given enrichment rules scoped to the cluster MEID.
// It tries the legacy schema first; if that schema is unavailable (404) it falls back to the new schema.
// BackupEnrichmentRulesOnTenant must run first.
func CreateEnrichmentRuleOnTenant(secretConfig tenant.Secret, rules ...metadataenrichment.Rule) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		state := getRulesState(ctx, t)

		settingsClient, err := tenant.BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		objectIDs, err := settingsClient.CreateLegacyEnrichmentRuleObject(ctx, state.scope, rules...)
		if core.IsNotFound(err) {
			t.Log("Legacy schema not available, falling back to new schema")

			objectIDs, err = settingsClient.CreateEnrichmentRuleObject(ctx, state.scope, rules...)
			require.NoError(t, err, "Could not create enrichment rule on tenant with new schema either. Please follow comment on ICP-1164 how to enable on tenant.")

			state.createdNewSchema = append(state.createdNewSchema, objectIDs...)
		}

		require.NoError(t, err, "Could not create enrichment rule on tenant")
		t.Logf("Created enrichment rule with objectId: %v (scope: %s)", objectIDs, state.scope)

		return ctx
	}
}

// RestoreEnrichmentRulesOnTenant removes the enrichment rules created by the test and restores the legacy object saved by BackupEnrichmentRulesOnTenant.
// Nothing is touched if no backup was made, to avoid deleting rules that do not belong to the test.
func RestoreEnrichmentRulesOnTenant(secretConfig tenant.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		state, ok := ctx.Value(rulesStateKey{}).(*rulesState)
		if !ok {
			t.Log("No enrichment rule backup found, skipping restore")

			return ctx
		}

		settingsClient, err := tenant.BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		// The legacy backup was removed in the setup, so whatever exists now was created by the test.
		if testLegacy := getLegacyRuleObject(ctx, t, settingsClient, state.scope); testLegacy != nil {
			deleteObject(ctx, t, settingsClient, testLegacy.ObjectID)
		}

		for _, objectID := range state.createdNewSchema {
			deleteObject(ctx, t, settingsClient, objectID)
		}

		if state.legacyBackup != nil {
			objectID, err := settingsClient.CreateRawObject(ctx, tenant.LegacyMetadataEnrichmentSchemaID, state.scope, state.legacyBackup)
			require.NoError(t, err, "Could not restore legacy enrichment rule object: %s", string(state.legacyBackup))
			t.Logf("Restored legacy enrichment rule object: %s", objectID)
		}

		return ctx
	}
}

// CheckEnrichmentRuleInDynaKubeStatus asserts that the DynaKube status contains an enrichment rule matching the expected type, source, and target.
func CheckEnrichmentRuleInDynaKubeStatus(dk *dynakube.DynaKube, expected ...metadataenrichment.Rule) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, dk))

		rules := dk.Status.MetadataEnrichment.Rules
		assert.NotEmpty(t, rules, "expected enrichment rules in DynaKube status, got none")

		for _, expect := range expected {
			if !slices.Contains(rules, expect) {
				t.Errorf("enrichment rule not found in DynaKube status: want %+v, got %+v", expected, rules)
			}
		}

		return ctx
	}
}
