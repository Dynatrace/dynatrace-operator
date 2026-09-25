// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

//go:build e2e

package codemodules

import (
	"bytes"
	"context"
	"io"
	"path"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube/oneagent"
	dtcsi "github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi"
	oacommon "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/features/cloudnative"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/csi"
	dynakubeComponents "github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/dynakube"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/components/operator"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8sdaemonset"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8snamespace"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/kubernetes/objects/k8spod"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/registry"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/sample"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/shell"
	"github.com/Dynatrace/dynatrace-operator/test/e2e/helpers/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/e2e-framework/third_party/helm"
)

const (
	cleanupPeriod             = "1s"
	cleanupStartedLogLine     = "running CSI filesystem cleanup"
	cleanupSettleDuration     = 30 * time.Second
	cleanupRunTimeout         = 10 * time.Minute
	cleanupPollInterval       = 15 * time.Second
	codeModuleDownloadTimeout = 10 * time.Minute
)

// CleanupKeepsMountedCodeModules verifies that upgrading the code module version does not take the
// agent files away from pods that are already injected and still running.
//
// This reproduces ICP-9640/ICP-9922, where the CSI garbage collection deleted the code module
// directory that running pods still used as the lower dir of their overlay mount. The agent then
// failed as soon as it had to read a file that was not already in the dentry cache.
//
// The assertion deliberately only looks at what an injected pod can see, not at any path inside the
// CSI filesystem. Those paths are what changed when the bug was introduced, so a test coupled to
// them would have to be rewritten by the very change that breaks the behavior.
func CleanupKeepsMountedCodeModules(t *testing.T) features.Feature {
	builder := features.New("cloudnative-codemodules-cleanup-keeps-mounted")
	secretConfig := tenant.GetSingleTenantSecret(t)

	previousImage := registry.GetPreviousCodeModulesImageTagURI(t)
	latestImage := registry.GetLatestCodeModulesImageTagURI(t)

	// Shared between the steps, filled in while the feature runs.
	agentFiles := &agentFilesSnapshot{}

	testDynakube := newAppMonDynakube(secretConfig.APIURL, previousImage)

	labels := testDynakube.OneAgent().GetNamespaceSelector().MatchLabels
	sampleNamespace := *k8snamespace.New("codemodules-cleanup-sample", k8snamespace.WithLabels(labels))

	sampleApp := sample.NewApp(t, &testDynakube,
		sample.WithNamespace(sampleNamespace),
	)

	builder.WithSetup("install operator with fast cleanup period", helpers.ToFeatureFunc(
		operator.InstallLocal(true, helm.WithArgs("--set", "csidriver.cleanupPeriod="+cleanupPeriod)), true))

	builder.Assess("create sample namespace", sampleApp.InstallNamespace())

	dynakubeComponents.Install(builder, &secretConfig, testDynakube)

	builder.Assess("install sample app", sampleApp.Install())
	cloudnative.AssessSampleInitContainers(builder, sampleApp)

	builder.Assess("remember the agent files of the injected pod", recordAgentFiles(sampleApp, agentFiles))

	dynakubeComponents.Update(builder, newAppMonDynakube(secretConfig.APIURL, latestImage))

	builder.Assess("new codemodule has been downloaded", waitForNewCodeModule(testDynakube, agentFiles))
	builder.Assess("two codemodules are present before cleanup", assertCodeModuleCount(testDynakube, agentFiles, 2))

	builder.Assess("record cleanup runs before triggering a reconcile", recordCleanupRunsBefore(testDynakube, agentFiles))
	dynakubeComponents.TriggerReconciliationWithoutWait(builder, testDynakube)
	builder.Assess("garbage collection ran after the upgrade", waitForCleanupRun(testDynakube, agentFiles))

	// The actual regression check. Same pod, never restarted, so it still uses the old code module.
	builder.Assess("injected pod can still read all its agent files", agentFilesAreUnchanged(testDynakube, sampleApp, agentFiles))

	// Guards against the opposite mistake: a garbage collection that keeps everything forever would
	// pass the check above.
	builder.Assess("uninstall sample app", sampleApp.Uninstall())

	builder.Assess("record cleanup runs before triggering a reconcile", recordCleanupRunsBefore(testDynakube, agentFiles))
	dynakubeComponents.TriggerReconciliationWithoutWait(builder, testDynakube)
	builder.Assess("garbage collection ran after the sample app is gone", waitForCleanupRun(testDynakube, agentFiles))
	builder.Assess("unused codemodule is cleaned up eventually", codeModuleIsRemoved(testDynakube, agentFiles))

	builder.WithTeardown("switch back to regular operator installation", restoreOperatorInstallation())

	return builder.Feature()
}

func newAppMonDynakube(apiUrl, codeModulesImage string) dynakube.DynaKube {
	return *dynakubeComponents.New(
		dynakubeComponents.WithName("codemodules-cleanup"),
		dynakubeComponents.WithApplicationMonitoringSpec(&oneagent.ApplicationMonitoringSpec{}),
		dynakubeComponents.WithCodeModulesImage(codeModulesImage),
		dynakubeComponents.WithNameBasedOneAgentNamespaceSelector(),
		dynakubeComponents.WithNameBasedMetadataEnrichmentNamespaceSelector(),
		dynakubeComponents.WithAPIURL(apiUrl),
	)
}

func restoreOperatorInstallation() features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		if err := operator.InstallViaHelm("", true, helm.WithArgs("--set", "csidriver.cleanupPeriod=")); err != nil {
			t.Logf("failed to restore the cleanup period: %v", err)
		}

		return ctx
	}
}

type agentFilesSnapshot struct {
	manifestChecksum    string
	numberOfAgentFiles  int
	injectedPodNodeName string

	cleanupRunsBeforeTrigger int
	codeModuleDir            string
}

// readAgentFiles lists every agent file the pod can see and checksums that listing.
func readAgentFiles(ctx context.Context, t *testing.T, resource *resources.Resources, pod corev1.Pod, container string) (string, int) {
	t.Helper()

	findFiles := "find " + oacommon.DefaultInstallPath + " -type f | sort"
	command := shell.Shell(shell.Command{findFiles + " | sha256sum; " + findFiles + " | wc -l"})

	result, err := k8spod.Exec(ctx, resource, pod, container, command...)
	require.NoError(t, err)

	output := strings.Fields(result.StdOut.String())
	require.NotEmpty(t, output, "no output while listing the agent files")

	count, err := strconv.Atoi(output[len(output)-1])
	require.NoError(t, err)

	return output[0], count
}

// readCodeModuleDir reports the lower dir of the overlay the pod is injected with.
func readCodeModuleDir(ctx context.Context, t *testing.T, resource *resources.Resources, pod corev1.Pod, container string) string {
	t.Helper()

	command := shell.Shell(shell.Command{"grep ' " + oacommon.DefaultInstallPath + " ' /proc/self/mounts"})

	result, err := k8spod.Exec(ctx, resource, pod, container, command...)
	require.NoError(t, err)

	for option := range strings.SplitSeq(result.StdOut.String(), ",") {
		if lowerDir, found := strings.CutPrefix(option, "lowerdir="); found {
			return lowerDir
		}
	}

	require.Failf(t, "no lowerdir found", "mount line: %s", result.StdOut.String())

	return ""
}

func recordAgentFiles(sampleApp *sample.App, snapshot *agentFilesSnapshot) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := sampleApp.GetPod(ctx, t, resource)

		snapshot.manifestChecksum, snapshot.numberOfAgentFiles = readAgentFiles(ctx, t, resource, pod, sampleApp.ContainerName())
		snapshot.codeModuleDir = readCodeModuleDir(ctx, t, resource, pod, sampleApp.ContainerName())
		snapshot.injectedPodNodeName = pod.Spec.NodeName

		require.Positive(t, snapshot.numberOfAgentFiles, "the injected pod has no agent files to begin with")
		t.Logf("pod %s on %s sees %d agent files, mounted from %s",
			pod.Name, snapshot.injectedPodNodeName, snapshot.numberOfAgentFiles, snapshot.codeModuleDir)

		return ctx
	}
}

func agentFilesAreUnchanged(dk dynakube.DynaKube, sampleApp *sample.App, snapshot *agentFilesSnapshot) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := sampleApp.GetPod(ctx, t, resource)

		manifest, count := readAgentFiles(ctx, t, resource, pod, sampleApp.ContainerName())

		assert.Equal(t, snapshot.numberOfAgentFiles, count,
			"the injected pod lost agent files after the codemodule upgrade, it was mounted from %s", snapshot.codeModuleDir)
		assert.Equal(t, snapshot.manifestChecksum, manifest,
			"the agent files of the injected pod changed after the codemodule upgrade, it was mounted from %s", snapshot.codeModuleDir)

		if t.Failed() {
			// The CSI driver runs in the namespace of the DynaKube, not in the one of the sample app.
			logCodeModulesDir(ctx, t, envConfig, dk.Namespace, snapshot.injectedPodNodeName)
		}

		return ctx
	}
}

// waitForNewCodeModule waits until the provisioners downloaded a code module other than the one the
// sample app is mounted with, which is when the old one becomes a candidate for the cleanup.
//
// ImageHasBeenDownloaded cannot be used for this: it also accepts the "agent already installed" log
// line, which the first code module already put into the log, so it would return right away.
func waitForNewCodeModule(dk dynakube.DynaKube, snapshot *agentFilesSnapshot) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := csiPodOnNode(ctx, t, resource, dk.Namespace, snapshot.injectedPodNodeName)

		err := wait.For(func(ctx context.Context) (done bool, err error) {
			latest, readErr := readLatestCodeModule(ctx, resource, pod, dk.Name)
			if readErr != nil {
				t.Logf("failed to read the latest codemodule link on %s, retrying: %v", pod.Name, readErr)

				return false, nil
			}

			if latest != snapshot.codeModuleDir {
				t.Logf("latest codemodule on %s moved to %s", pod.Name, latest)

				return true, nil
			}

			t.Logf("wait for the latest codemodule on %s to move away from %s", pod.Name, latest)

			return false, nil
		}, wait.WithTimeout(codeModuleDownloadTimeout), wait.WithInterval(cleanupPollInterval))
		require.NoError(t, err)

		return ctx
	}
}

// readLatestCodeModule reads the symlink from /data/_dynakube/<name>/latest-codemodule
func readLatestCodeModule(ctx context.Context, resource *resources.Resources, pod corev1.Pod, dynakubeName string) (string, error) {
	latestLink := path.Join(dataPath, dtcsi.SharedDynaKubesDir, dynakubeName, "latest-codemodule")
	command := shell.Shell(shell.Command{"readlink " + latestLink})

	result, err := k8spod.Exec(ctx, resource, pod, provisionerContainerName, command...)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(result.StdOut.String()), nil
}

// csiPodOnNode returns the CSI driver pod running on the given node. Only that one provisions and
// cleans up the code modules of pods on that node, so waiting for any other would just cost time.
func csiPodOnNode(ctx context.Context, t *testing.T, resource *resources.Resources, namespace, nodeName string) corev1.Pod {
	t.Helper()

	var csiPod corev1.Pod

	err := k8sdaemonset.NewQuery(ctx, resource, client.ObjectKey{
		Name:      csi.DaemonSetName,
		Namespace: namespace,
	}).ForEachPod(func(pod corev1.Pod) {
		if pod.Spec.NodeName == nodeName {
			csiPod = pod
		}
	})
	require.NoError(t, err)
	require.NotEmpty(t, csiPod.Name, "no csi driver pod found on node %s", nodeName)

	return csiPod
}

func listCodeModules(ctx context.Context, resource *resources.Resources, pod corev1.Pod) ([]string, error) {
	listCommand := shell.ListDirectory(dataPath + dtcsi.SharedAgentBinDir)

	result, err := k8spod.Exec(ctx, resource, pod, provisionerContainerName, listCommand...)
	if err != nil {
		return nil, err
	}

	return strings.Fields(result.StdOut.String()), nil
}

func assertCodeModuleCount(dk dynakube.DynaKube, snapshot *agentFilesSnapshot, expected int) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := csiPodOnNode(ctx, t, resource, dk.Namespace, snapshot.injectedPodNodeName)

		codeModules, err := listCodeModules(ctx, resource, pod)
		require.NoError(t, err)

		t.Logf("codemodules on %s: %v", pod.Name, codeModules)
		assert.Len(t, codeModules, expected, "unexpected number of codemodules in /data/codemodules on %s", pod.Name)

		return ctx
	}
}

// codeModuleIsRemoved checks that the code module the sample app used is gone once nothing mounts
// it anymore. This is the only step that looks into the CSI filesystem, because "it was deleted" is
// not observable from a pod that does not exist anymore.
func codeModuleIsRemoved(dk dynakube.DynaKube, snapshot *agentFilesSnapshot) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		pod := csiPodOnNode(ctx, t, resource, dk.Namespace, snapshot.injectedPodNodeName)

		err := wait.For(func(ctx context.Context) (done bool, err error) {
			codeModules, listErr := listCodeModules(ctx, resource, pod)
			if listErr != nil {
				t.Logf("failed to list the codemodules on %s, retrying: %v", pod.Name, listErr)

				return false, nil
			}

			return !slices.Contains(codeModules, codeModuleName(snapshot.codeModuleDir)), nil
		}, wait.WithTimeout(cleanupRunTimeout), wait.WithInterval(cleanupPollInterval))
		assert.NoError(t, err, "codemodule %s was never cleaned up on %s", snapshot.codeModuleDir, pod.Name)

		return ctx
	}
}

// recordCleanupRunsBefore captures the baseline waitForCleanupRun waits to see exceeded. Must run
// as its own step before TriggerReconciliation, see the comment on cleanupRunsBeforeTrigger.
func recordCleanupRunsBefore(dk dynakube.DynaKube, snapshot *agentFilesSnapshot) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())
		require.NoError(t, err)

		pod := csiPodOnNode(ctx, t, resource, dk.Namespace, snapshot.injectedPodNodeName)

		runs, err := countCleanupRuns(ctx, clientset, pod)
		require.NoError(t, err)

		snapshot.cleanupRunsBeforeTrigger = runs

		return ctx
	}
}

// waitForCleanupRun waits until the sample app's node's CSI driver started another garbage
// collection run, counting from the baseline recordCleanupRunsBefore captured earlier. Without this
// the assertions could run before any garbage collection happened and pass for the wrong reason.
func waitForCleanupRun(dk dynakube.DynaKube, snapshot *agentFilesSnapshot) features.Func {
	return func(ctx context.Context, t *testing.T, envConfig *envconf.Config) context.Context {
		resource := envConfig.Client().Resources()
		clientset, err := kubernetes.NewForConfig(resource.GetConfig())
		require.NoError(t, err)

		pod := csiPodOnNode(ctx, t, resource, dk.Namespace, snapshot.injectedPodNodeName)
		runsBefore := snapshot.cleanupRunsBeforeTrigger

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			runs, countErr := countCleanupRuns(ctx, clientset, pod)
			if countErr != nil {
				// A blip while talking to the api server should not end a test that runs for
				// minutes, the next poll can try again.
				t.Logf("failed to read the provisioner log of %s, retrying: %v", pod.Name, countErr)

				return false, nil
			}

			t.Logf("wait for a cleanup run on %s, %d of %d", pod.Name, runs, runsBefore+1)

			return runs > runsBefore, nil
		}, wait.WithTimeout(cleanupRunTimeout), wait.WithInterval(cleanupPollInterval))
		require.NoError(t, err)

		// The run is logged when it starts, give it a moment to finish.
		time.Sleep(cleanupSettleDuration)

		return ctx
	}
}

func countCleanupRuns(ctx context.Context, clientset *kubernetes.Clientset, pod corev1.Pod) (int, error) {
	logStream, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: provisionerContainerName,
	}).Stream(ctx)
	if err != nil {
		return 0, err
	}

	defer func() { _ = logStream.Close() }()

	buffer := new(bytes.Buffer)
	if _, err := io.Copy(buffer, logStream); err != nil {
		return 0, err
	}

	return strings.Count(buffer.String(), cleanupStartedLogLine), nil
}

// logCodeModulesDir dumps what the provisioners have on disk, to make a failed assertion easier to
// understand. It is diagnostics only, nothing asserts on it.
func logCodeModulesDir(ctx context.Context, t *testing.T, envConfig *envconf.Config, namespace, nodeName string) {
	t.Helper()

	resource := envConfig.Client().Resources()
	pod := csiPodOnNode(ctx, t, resource, namespace, nodeName)

	codeModules, err := listCodeModules(ctx, resource, pod)
	if err == nil {
		t.Logf("codemodules on %s: %v", pod.Name, codeModules)
	}
}

func codeModuleName(codeModuleDir string) string {
	return codeModuleDir[strings.LastIndex(codeModuleDir, "/")+1:]
}
