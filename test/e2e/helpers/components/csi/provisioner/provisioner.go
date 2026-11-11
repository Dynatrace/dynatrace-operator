// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package provisioner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
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

// Installer is the codemodule installer that the provisioner is expected to use.
type Installer string

const (
	ImageInstaller Installer = "image"
	JobInstaller   Installer = "job"
)

const (
	installTimeout = 5 * time.Minute

	// messages logged by the codemodule installers, the image is part of the same line.
	installedMessage        = "installed agent via"
	alreadyInstalledMessage = "agent already installed"
)

// InstalledImage asserts that the provisioner of every CSI pod installed the image for the DynaKube using the given installer.
// The CSI pods outlive the tests, so only the log lines of this DynaKube since its creation are considered.
// The image is matched as a whole log value, so a longer image reference with the same prefix does not match.
func InstalledImage(dk *dynakube.DynaKube, image string, installer Installer) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		require.NotEmpty(t, image)

		resources := envConfig.Client().Resources()

		var current dynakube.DynaKube
		require.NoError(t, resources.Get(ctx, dk.Name, dk.Namespace, &current))

		since := current.CreationTimestamp.Time
		imageValue := fmt.Sprintf(`"image":"%s"`, image)
		installerValue := fmt.Sprintf(`"logger":"csi-provisioner.%s"`, installer)
		dynakubeValue := fmt.Sprintf(`"DynaKube":{"name":"%s","namespace":"%s"}`, dk.Name, dk.Namespace)

		err := k8sdaemonset.NewQuery(ctx, resources, client.ObjectKey{
			Name:      csi.DaemonSetName,
			Namespace: dk.Namespace,
		}).ForEachPod(func(pod *corev1.Pod) {
			// ReadLog fails the test on error, which would hide the assertion message below on timeout.
			err := wait.For(func(context.Context) (bool, error) {
				provisionerLog := logs.ReadLogSince(ctx, t, envConfig, pod.Namespace, pod.Name, csi.ProvisionerContainerName, since)

				for _, message := range []string{installedMessage, alreadyInstalledMessage} {
					if logs.FindLineContainingText(provisionerLog, message, imageValue, installerValue, dynakubeValue) != "" {
						return true, nil
					}
				}

				t.Logf("provisioner log of %s has no installation message for image %s via %s installer yet, retrying", pod.Name, image, installer)

				return false, nil
			}, wait.WithTimeout(installTimeout))
			require.NoError(t, err, "provisioner of %s did not install image %s via %s installer", pod.Name, image, installer)
		})
		require.NoError(t, err)

		return ctx
	}
}
