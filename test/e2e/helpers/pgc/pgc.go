// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package pgc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/shell"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const FilePath = "/mnt/volume_storage_mount/opt/agent/conf/" + bootstrapperconfig.DeclarativeInputFileName

func WaitForFileInAllPods(dk dynakube.DynaKube) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		r := envConfig.Client().Resources()

		err := wait.For(func(ctx context.Context) (bool, error) {
			return allPodsHaveFile(ctx, r, dk)
		}, wait.WithTimeout(10*time.Minute), wait.WithInterval(30*time.Second))

		require.NoError(t, err)

		return ctx
	}
}

func allPodsHaveFile(ctx context.Context, r *resources.Resources, dk dynakube.DynaKube) (bool, error) {
	q := k8sdaemonset.NewQuery(ctx, r, client.ObjectKey{
		Name:      dk.OneAgent().GetDaemonsetName(),
		Namespace: dk.Namespace,
	})

	allFound := true
	podCount := 0
	var firstExecErr error

	err := q.ForEachPod(func(pod corev1.Pod) {
		podCount++

		if !allFound || len(pod.Spec.Containers) == 0 {
			allFound = false

			return
		}

		result, execErr := k8spod.Exec(ctx, r, pod, pod.Spec.Containers[0].Name, shell.Shell(shell.Exists(FilePath))...)
		if execErr != nil {
			if firstExecErr == nil {
				firstExecErr = execErr
			}

			allFound = false

			return
		}

		if !strings.Contains(result.StdOut.String(), "found") {
			allFound = false
		}
	})

	if err != nil {
		return false, err
	}

	if firstExecErr != nil {
		return false, firstExecErr
	}

	return podCount > 0 && allFound, nil
}
