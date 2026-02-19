//go:build e2e

package cloudnative

import (
	"context"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/activegate"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/shell"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

const (
	oneAgentCustomPemPath      = "/var/lib/dynatrace/oneagent/agent/customkeys/custom.pem"
	oneAgentCustomProxyPemPath = "/var/lib/dynatrace/oneagent/agent/customkeys/custom_proxy.pem"
	activeGateRootCAPath       = "/var/lib/dynatrace/secrets/rootca/rootca.pem"
)

func AssessSampleContainer(builder *features.FeatureBuilder, sampleApp *sample.App, agCrtFunc func() []byte, trustedCAs []byte) {
	builder.Assess("certificates are propagated to sample apps containers", checkSampleContainer(sampleApp, agCrtFunc, trustedCAs))
}

func checkSampleContainer(sampleApp *sample.App, agCrtFunc func() []byte, trustedCAs []byte) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		pods := sampleApp.GetPods(ctx, t, resources)
		require.NotEmpty(t, pods.Items)

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			require.NotEmpty(t, pod.Spec.Containers)

			certs := string(agCrtFunc()) + "\n" + string(trustedCAs)

			if string(agCrtFunc()) == "" && string(trustedCAs) == "" {
				checkFileNotFound(ctx, t, resources, pod, sampleApp.ContainerName(), oneAgentCustomPemPath)
			} else {
				checkFileContents(ctx, t, resources, pod, sampleApp.ContainerName(), oneAgentCustomPemPath, certs)
			}
			if string(trustedCAs) == "" {
				checkFileNotFound(ctx, t, resources, pod, sampleApp.ContainerName(), oneAgentCustomProxyPemPath)
			} else {
				checkFileContents(ctx, t, resources, pod, sampleApp.ContainerName(), oneAgentCustomProxyPemPath, string(trustedCAs))
			}
		}

		return ctx
	}
}

func AssessOneAgentContainer(builder *features.FeatureBuilder, agCrtFunc func() []byte, trustedCAs []byte) {
	builder.Assess("certificates are propagated to OneAgent containers", checkOneAgentContainer(agCrtFunc, trustedCAs))
}

func checkOneAgentContainer(agCrtFunc func() []byte, trustedCAs []byte) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		// TODO: when OneAgent ticket is done, probably the same pem files as in case of sample container

		return ctx
	}
}

func AssessActiveGateContainer(builder *features.FeatureBuilder, dk *dynakube.DynaKube, trustedCAs []byte) {
	builder.Assess("certificates are propagated to ActiveGate container", checkActiveGateContainer(dk, trustedCAs))
}

func checkActiveGateContainer(dk *dynakube.DynaKube, trustedCAs []byte) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resources := envConfig.Client().Resources()

		var activeGatePod corev1.Pod
		require.NoError(t, resources.WithNamespace(dk.Namespace).Get(ctx, activegate.GetActiveGatePodName(dk, "activegate"), dk.Namespace, &activeGatePod))

		require.NotNil(t, activeGatePod.Spec)
		require.NotEmpty(t, activeGatePod.Spec.Containers)

		if string(trustedCAs) == "" {
			checkFileNotFound(ctx, t, resources, activeGatePod, "activegate", activeGateRootCAPath)
		} else {
			checkFileContents(ctx, t, resources, activeGatePod, "activegate", activeGateRootCAPath, string(trustedCAs))
		}

		return ctx
	}
}

func checkFileContents(ctx context.Context, t *testing.T, testResources *resources.Resources, testPod corev1.Pod, containerName string, filename string, certificates string) { //nolint:revive
	catCommand := shell.Shell(shell.Cat(filename))
	executionResult, err := k8spod.Exec(ctx, testResources, testPod, containerName, catCommand...)

	require.NoError(t, err)

	stdOut := executionResult.StdOut.String()
	stdErr := executionResult.StdErr.String()

	assert.Equal(t, certificates, stdOut)
	assert.Empty(t, stdErr)
}

func checkFileNotFound(ctx context.Context, t *testing.T, testResources *resources.Resources, testPod corev1.Pod, containerName string, filename string) { //nolint:revive
	existsCommand := shell.Shell(shell.Exists(filename))
	executionResult, err := k8spod.Exec(ctx, testResources, testPod, containerName, existsCommand...)

	require.NoError(t, err)

	stdOut := executionResult.StdOut.String()
	stdErr := executionResult.StdErr.String()

	assert.Empty(t, stdOut)
	assert.Empty(t, stdErr)
}
