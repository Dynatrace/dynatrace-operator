// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package csi

import (
	"context"
	"fmt"
	"strings"

	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	DaemonSetName = "dynatrace-oneagent-csi-driver"

	ProvisionerContainerName = "provisioner"
)

func CleanUpEachPod(namespace string) env.Func {
	return func(ctx context.Context, envConfig *envconf.Config) (context.Context, error) {
		resource := envConfig.Client().Resources()

		err := k8sdaemonset.NewQuery(ctx, resource, client.ObjectKey{
			Name:      DaemonSetName,
			Namespace: namespace,
		}).ForEachPod(cleanUpPodConsumer(ctx, resource))
		if k8serrors.IsNotFound(err) {
			return ctx, nil
		}

		return ctx, err
	}
}

func WaitForDaemonset(namespace string) env.Func {
	return k8sdaemonset.WaitFor(DaemonSetName, namespace)
}

func cleanUpPodConsumer(ctx context.Context, resource *resources.Resources) k8sdaemonset.PodConsumer {
	return func(p *corev1.Pod) {
		// Only remove the content: the data directory is a mount point and can't be removed itself.
		// The globs are expanded by the shell of the debug container: the CSI image doesn't contain a shell.
		dataPath := strings.TrimSuffix(dtcsi.DataPath, "/")
		removeContent := fmt.Sprintf("rm -rf %[1]s/* %[1]s/.[!.]* %[1]s/..?*", dataPath)

		_, err := k8spod.ExecInDebugContainer(ctx, resource, p, "server", "sh", "-c", removeContent)
		if err != nil {
			// a leftover data directory makes later assertions about the used disk space fail in a confusing way
			klog.ErrorS(err, "failed to clean up the CSI data directory", "pod", p.Name, "namespace", p.Namespace)
		}
	}
}
