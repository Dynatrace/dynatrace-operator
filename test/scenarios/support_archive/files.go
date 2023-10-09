//go:build e2e

package support_archive

import (
	"context"
	"fmt"
	"testing"

	support_archive2 "github.com/Dynatrace/dynatrace-operator/cmd/support_archive"
	edgeconnectv1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/functional"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/csi"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/oneagent"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/components/operator"
	e2ewebhook "github.com/Dynatrace/dynatrace-operator/test/helpers/components/webhook"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/pod"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/replicaset"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/service"
	"github.com/Dynatrace/dynatrace-operator/test/helpers/kubeobjects/statefulset"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

type requiredFiles struct {
	t              *testing.T
	ctx            context.Context
	resources      *resources.Resources
	dynakube       dynatracev1beta1.DynaKube
	edgeconnect    edgeconnectv1beta1.EdgeConnect
	collectManaged bool
}

func newRequiredFiles(t *testing.T, ctx context.Context, resources *resources.Resources, customResources CustomResources, collectManaged bool) requiredFiles {
	return requiredFiles{
		t:              t,
		ctx:            ctx,
		resources:      resources,
		dynakube:       customResources.dynakube,
		edgeconnect:    customResources.edgeconnect,
		collectManaged: collectManaged,
	}
}

func (r requiredFiles) collectRequiredFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles, support_archive2.OperatorVersionFileName)
	requiredFiles = append(requiredFiles, support_archive2.TroublshootOutputFileName)
	requiredFiles = append(requiredFiles, support_archive2.SupportArchiveOutputFileName)
	requiredFiles = append(requiredFiles, r.getRequiredPodFiles(kubeobjects.AppNameLabel, true)...)
	requiredFiles = append(requiredFiles, r.getRequiredPodFiles(kubeobjects.AppManagedByLabel, r.collectManaged)...)
	requiredFiles = append(requiredFiles, r.getRequiredReplicaSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredServiceFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredWorkloadFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredNamespaceFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredDynaKubeFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredEdgeConnectFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredStatefulSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredDaemonSetFiles()...)
	return requiredFiles
}

func (r requiredFiles) getRequiredPodFiles(labelKey string, collectManaged bool) []string {
	pods := pod.List(r.t, r.ctx, r.resources, r.dynakube.Namespace)
	requiredFiles := make([]string, 0)

	podList := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		label, ok := podItem.Labels[labelKey]
		return ok && label == operator.DeploymentName
	})

	for _, operatorPod := range podList {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/pod/%s%s",
				support_archive2.ManifestsDirectoryName,
				operatorPod.Namespace, operatorPod.Name,
				support_archive2.ManifestsFileExtension))
		if collectManaged && (labelKey == "app.kubernetes.io/managed-by" || labelKey == "app.kubernetes.io/name") {
			for _, container := range operatorPod.Spec.Containers {
				requiredFiles = append(requiredFiles,
					fmt.Sprintf("%s/%s/%s.log", support_archive2.LogsDirectoryName, operatorPod.Name, container.Name))
			}
		}
	}
	return requiredFiles
}

func (r requiredFiles) getRequiredReplicaSetFiles() []string {
	replicaSets := replicaset.List(r.t, r.ctx, r.resources, r.dynakube.Namespace)
	requiredFiles := make([]string, 0)
	for _, replicaSet := range replicaSets.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/replicaset/%s%s",
				support_archive2.ManifestsDirectoryName,
				replicaSet.Namespace, replicaSet.Name,
				support_archive2.ManifestsFileExtension))
	}
	return requiredFiles
}

func (r requiredFiles) getRequiredStatefulSetFiles() []string {
	statefulSet, err := statefulset.NewQuery(r.ctx, r.resources, client.ObjectKey{
		Namespace: r.dynakube.Namespace,
		Name:      "dynakube-activegate"}).Get()
	require.NoError(r.t, err)
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/statefulset/%s%s",
			support_archive2.ManifestsDirectoryName,
			statefulSet.Namespace, statefulSet.Name,
			support_archive2.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredDaemonSetFiles() []string {
	oneagentDaemonSet, err := oneagent.Get(r.ctx, r.resources, r.dynakube)
	require.NoError(r.t, err)
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/daemonset/%s%s",
			support_archive2.ManifestsDirectoryName,
			oneagentDaemonSet.Namespace,
			oneagentDaemonSet.Name,
			support_archive2.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredServiceFiles() []string {
	services := service.List(r.t, r.ctx, r.resources, r.dynakube.Namespace)
	requiredFiles := make([]string, 0)
	for _, requiredService := range services.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/service/%s%s",
				support_archive2.ManifestsDirectoryName,
				requiredService.Namespace,
				requiredService.Name,
				support_archive2.ManifestsFileExtension))
	}
	return requiredFiles
}

func (r requiredFiles) getRequiredWorkloadFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive2.ManifestsDirectoryName,
			r.dynakube.Namespace,
			"deployment",
			operator.DeploymentName,
			support_archive2.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive2.ManifestsDirectoryName,
			r.dynakube.Namespace,
			"deployment",
			e2ewebhook.DeploymentName,
			support_archive2.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive2.ManifestsDirectoryName,
			r.dynakube.Namespace,
			"daemonset",
			csi.DaemonSetName,
			support_archive2.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive2.ManifestsDirectoryName,
			r.edgeconnect.Namespace,
			"deployment",
			r.edgeconnect.Name,
			support_archive2.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredNamespaceFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/namespace-%s%s",
			support_archive2.ManifestsDirectoryName,
			r.dynakube.Namespace,
			r.dynakube.Namespace,
			support_archive2.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/namespace-%s%s",
			support_archive2.ManifestsDirectoryName,
			support_archive2.InjectedNamespacesManifestsDirectoryName,
			testAppNameInjected,
			support_archive2.ManifestsFileExtension))
	return requiredFiles
}

func (r requiredFiles) getRequiredDynaKubeFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive2.ManifestsDirectoryName,
			r.dynakube.Namespace,
			"dynakube",
			r.dynakube.Name,
			support_archive2.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredEdgeConnectFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive2.ManifestsDirectoryName,
			r.edgeconnect.Namespace,
			"edgeconnect",
			r.edgeconnect.Name,
			support_archive2.ManifestsFileExtension))

	return requiredFiles
}
