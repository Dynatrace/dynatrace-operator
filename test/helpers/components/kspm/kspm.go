//go:build e2e

package kspm

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	dtsettings "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/settings"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	kubeSystemNamespace = "kube-system"
)

func BuildSettingsClient(secretConfig tenant.Secret) (dtsettings.APIClient, error) {
	dtClient, err := dtclient.NewClient(secretConfig.APIURL, secretConfig.APIToken, "", dtclient.SkipCertificateValidation(false))
	if err != nil {
		return nil, err
	}

	return dtClient.AsV2().Settings, nil
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
		err := resources.Get(ctx, kubeSystemNamespace, "", &kubeSystemNS)
		if err != nil {
			t.Logf("Could not get kube-system namespace: %v, skipping KSPM settings cleanup", err)

			return ctx
		}

		kubeSystemUUID := string(kubeSystemNS.UID)
		t.Logf("kube-system UUID: %s", kubeSystemUUID)

		settingsClient, err := BuildSettingsClient(secretConfig)
		if err != nil {
			t.Logf("Could not build settings client: %v, skipping KSPM settings cleanup", err)

			return ctx
		}

		k8sClusterME, err := settingsClient.GetK8sClusterME(ctx, kubeSystemUUID)
		if err != nil {
			t.Logf("Could not get K8s cluster MEID: %v, skipping KSPM settings cleanup", err)

			return ctx
		}

		if k8sClusterME.ID == "" {
			t.Log("No Kubernetes Cluster MEID found, skipping KSPM settings cleanup")

			return ctx
		}

		t.Logf("Found Kubernetes Cluster MEID: %s", k8sClusterME.ID)

		kspmSettings, err := settingsClient.GetKSPMSettings(ctx, k8sClusterME.ID)
		if err != nil {
			t.Logf("Could not query KSPM settings: %v, skipping cleanup", err)

			return ctx
		}

		if kspmSettings.TotalCount == 0 {
			t.Log("No existing KSPM settings found on tenant")

			return ctx
		}

		t.Logf("Found %d existing KSPM settings, attempting to delete them", kspmSettings.TotalCount)

		settingsWithIDs, err := settingsClient.GetKSPMSettings(ctx, k8sClusterME.ID)
		if err != nil {
			t.Logf("Could not get KSPM settings with IDs: %v, skipping cleanup", err)

			return ctx
		}

		for _, setting := range settingsWithIDs.Items {
			t.Log("Deleting KSPM setting")
			if err := settingsClient.DeleteSettings(ctx, setting.ObjectID); err != nil {
				t.Logf("Failed to delete KSPM setting: %v", err)
			} else {
				t.Log("Successfully deleted KSPM setting")
			}
		}

		return ctx
	}
}
