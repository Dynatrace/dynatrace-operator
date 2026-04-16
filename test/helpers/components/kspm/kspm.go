//go:build e2e

package kspm

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func BuildSettingsClient(secretConfig tenant.Secret) (dtsettings.APIClient, error) {
	dtClient, err := dynatrace.NewClient(
		dynatrace.WithBaseURL(secretConfig.APIURL),
		dynatrace.WithAPIToken(secretConfig.APIToken),
		dynatrace.WithPaasToken(""),
		dynatrace.WithSkipCertificateValidation(false))
	if err != nil {
		return nil, err
	}

	return dtClient.Settings, nil
}

func CheckKSPMSettingsExistOnTenant(secretConfig tenant.Secret, dk *dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		require.NoError(t, resources.Get(ctx, dk.Name, dk.Namespace, dk))

		require.NotEmpty(t, dk.Status.KubernetesClusterMEID, "KubernetesClusterMEID must be populated in DynaKube status")

		settingsClient, err := BuildSettingsClient(secretConfig)
		require.NoError(t, err)

		kspmSettings, err := settingsClient.GetKSPMSettings(ctx, dk.Status.KubernetesClusterMEID)
		require.NoError(t, err, "Failed to query KSPM settings from tenant")

		assert.Positive(t, kspmSettings.TotalCount, "KSPM settings should exist on the tenant")
		assert.True(t, kspmSettings.Items[0].Value.DatasetPipelineEnabled, "KSPM settings should have dataset pipeline enabled")

		return ctx
	}
}

func DeleteKSPMSettingsFromTenant(secretConfig tenant.Secret) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()
		var kubeSystemNS corev1.Namespace
		require.NoError(t, resources.Get(ctx, metav1.NamespaceSystem, "", &kubeSystemNS), "Could not get kube-system namespace")

		kubeSystemUUID := string(kubeSystemNS.UID)
		t.Logf("kube-system UUID: %s", kubeSystemUUID)

		settingsClient, err := BuildSettingsClient(secretConfig)
		require.NoError(t, err, "Could not build settings client")

		k8sClusterME, err := settingsClient.GetK8sClusterME(ctx, kubeSystemUUID)
		require.NoError(t, err, "Could not get K8s cluster MEID")

		if k8sClusterME.ID == "" {
			t.Log("No Kubernetes Cluster MEID found, skipping KSPM settings cleanup")

			return ctx
		}

		t.Logf("Found Kubernetes Cluster MEID: %s", k8sClusterME.ID)

		kspmSettings, err := settingsClient.GetKSPMSettings(ctx, k8sClusterME.ID)
		require.NoError(t, err, "Could not query KSPM settings")

		if kspmSettings.TotalCount == 0 {
			t.Log("No existing KSPM settings found on tenant")

			return ctx
		}

		t.Logf("Found %d existing KSPM settings, attempting to delete them", kspmSettings.TotalCount)

		for _, setting := range kspmSettings.Items {
			t.Logf("Deleting KSPM setting with ID: %s", setting.ObjectID)
			if err := settingsClient.DeleteSettings(ctx, setting.ObjectID); err != nil {
				t.Logf("Failed to delete KSPM setting with ID %s: %v", setting.ObjectID, err)
			} else {
				t.Logf("Successfully deleted KSPM setting with ID: %s", setting.ObjectID)
			}
		}

		return ctx
	}
}
