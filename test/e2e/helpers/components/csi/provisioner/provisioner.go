// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package provisioner

import (
	"context"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/logs"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	provisionerContainerName = "provisioner"
	installTimeout           = 5 * time.Minute

	// messages logged by the codemodule installers, the image is part of the same line.
	installedMessage        = "installed agent via"
	alreadyInstalledMessage = "agent already installed"
)

// InstalledImage asserts that the provisioner installed the image.
// It looks for the log line that mentions the image so "agent already installed" only counts
// if it was logged for that specific image.
func InstalledImage(namespace, image string) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NotEmpty(t, image)

		err := k8sdaemonset.NewQuery(ctx, envConfig.Client().Resources(), client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(func(pod *corev1.Pod) {
			err := wait.For(func(ctx context.Context) (bool, error) {
				provisionerLog := logs.ReadLog(ctx, t, envConfig, pod.Namespace, pod.Name, provisionerContainerName)

				for _, message := range []string{installedMessage, alreadyInstalledMessage} {
					if logs.FindLineContainingText(provisionerLog, message, image) != "" {
						return true, nil
					}
				}

				t.Logf("provisioner log of %s has no installation message for image %s yet, retrying", pod.Name, image)

				return false, nil
			}, wait.WithTimeout(installTimeout))
			require.NoError(t, err, "provisioner of %s did not install image %s", pod.Name, image)
		})
		require.NoError(t, err)

		return ctx
	}
}
