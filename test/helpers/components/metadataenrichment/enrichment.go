//go:build e2e

package metadataenrichment

import (
	"context"
	"errors"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/metadataenrichment"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/core"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func BuildSettingsClient(secretConfig tenant.Secret) (dtsettings.Client, error) {
	dtClient, err := dynatrace.NewClient(
		dynatrace.WithBaseURL(secretConfig.APIURL),
		dynatrace.WithAPIToken(secretConfig.TokensWithSettingsScope().APIToken),
		dynatrace.WithSkipCertificateValidation(false))
	if err != nil {
		return nil, err
	}

	return dtClient.Settings, nil
}

func getKubeSystemUUID(ctx context.Context, t *testing.T, envConfig *envconf.Config) string {
	t.Helper()

	var kubeSystemNS corev1.Namespace
	require.NoError(t, envConfig.Client().Resources().Get(ctx, metav1.NamespaceSystem, "", &kubeSystemNS))

	return string(kubeSystemNS.UID)
}

// EnsureKubernetesClusterMEID creates the builtin:cloud.kubernetes setting if not present,
// triggering the creation of the Kubernetes Cluster Monitored Entity on the tenant.
// It retries until the MEID is visible via the API.
func EnsureKubernetesClusterMEID(secretConfig tenant.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		settingsClient, err := BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		kubeSystemUUID := getKubeSystemUUID(ctx, t, envConfig)
		t.Logf("kube-system UUID: %s", kubeSystemUUID)

		_, err = settingsClient.CreateOrUpdateKubernetesSetting(ctx, "e2e-enrichment-test", kubeSystemUUID, "")
		require.NoError(t, err, "Could not create Kubernetes connection setting")

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

// CreateEnrichmentRuleOnTenant creates a single enrichment rule scoped to the cluster MEID.
// It tries the legacy schema first; if that schema is unavailable (404) it falls back to the new schema.
func CreateEnrichmentRuleOnTenant(secretConfig tenant.Secret, rule metadataenrichment.Rule) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		settingsClient, err := BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		kubeSystemUUID := getKubeSystemUUID(ctx, t, envConfig)

		k8sClusterME, err := settingsClient.GetK8sClusterME(ctx, kubeSystemUUID)
		require.NoError(t, err, "Could not get K8s cluster MEID")
		require.NotEmpty(t, k8sClusterME.ID, "Kubernetes Cluster MEID must exist before creating enrichment rules")

		objectID, err := settingsClient.CreateEnrichmentRule(ctx, dtsettings.LegacyMetadataEnrichmentSchemaID, k8sClusterME.ID, rule)
		if core.IsNotFound(err) {
			t.Logf("Legacy schema (%s) not available, falling back to new schema (%s)", dtsettings.LegacyMetadataEnrichmentSchemaID, dtsettings.MetadataEnrichmentSchemaID)

			objectID, err = settingsClient.CreateEnrichmentRule(ctx, dtsettings.MetadataEnrichmentSchemaID, k8sClusterME.ID, rule)
			require.NoError(t, err, "Could not create enrichment rule on tenant with new schema either. Please follow comment on ICP-1164 how to enable on tenant.")
		}

		require.NoError(t, err, "Could not create enrichment rule on tenant")
		t.Logf("Created enrichment rule with objectId: %s (scope: %s)", objectID, k8sClusterME.ID)

		return ctx
	}
}

// DeleteEnrichmentRulesFromTenant deletes all enrichment rule objects scoped to the cluster MEID.
// It tries the legacy schema first; if that schema is unavailable (404) it falls back to the new schema.
func DeleteEnrichmentRulesFromTenant(secretConfig tenant.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		settingsClient, err := BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		kubeSystemUUID := getKubeSystemUUID(ctx, t, envConfig)

		k8sClusterME, err := settingsClient.GetK8sClusterME(ctx, kubeSystemUUID)
		require.NoError(t, err, "Could not get K8s cluster MEID")

		if k8sClusterME.ID == "" {
			t.Log("No Kubernetes Cluster MEID found, skipping enrichment rules cleanup")

			return ctx
		}

		t.Logf("Deleting enrichment rules for MEID: %s", k8sClusterME.ID)

		objects, err := settingsClient.GetLegacyEnrichmentRuleObjects(ctx, k8sClusterME.ID)
		if core.IsNotFound(err) {
			t.Logf("Legacy schema (%s) not available, falling back to new schema (%s)", dtsettings.LegacyMetadataEnrichmentSchemaID, dtsettings.MetadataEnrichmentSchemaID)

			objects, err = settingsClient.GetEnrichmentRuleObjects(ctx, k8sClusterME.ID)
		}

		require.NoError(t, err, "Could not list enrichment rule objects")

		if len(objects) == 0 {
			t.Log("No enrichment rules found on tenant, nothing to clean up")

			return ctx
		}

		t.Logf("Found %d enrichment rule(s), deleting", len(objects))

		for _, obj := range objects {
			if err := settingsClient.DeleteSettings(ctx, obj.ObjectID); err != nil {
				t.Logf("Failed to delete enrichment rule %s: %v", obj.ObjectID, err)
			} else {
				t.Logf("Deleted enrichment rule: %s", obj.ObjectID)
			}
		}

		return ctx
	}
}

// CheckEnrichmentRuleInDynaKubeStatus asserts that the DynaKube status contains an enrichment rule matching the expected type, source, and target.
func CheckEnrichmentRuleInDynaKubeStatus(dk *dynakube.DynaKube, expected metadataenrichment.Rule) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NoError(t, envConfig.Client().Resources().Get(ctx, dk.Name, dk.Namespace, dk))

		rules := dk.Status.MetadataEnrichment.Rules
		assert.NotEmpty(t, rules, "expected enrichment rules in DynaKube status, got none")

		for _, rule := range rules {
			if rule.Type == expected.Type && rule.Source == expected.Source && rule.Target == expected.Target {
				return ctx
			}
		}

		t.Errorf("enrichment rule not found in DynaKube status: want %+v, got %+v", expected, rules)

		return ctx
	}
}
