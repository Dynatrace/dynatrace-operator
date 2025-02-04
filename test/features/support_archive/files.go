//go:build e2e

package support_archive

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Dynatrace/dynatrace-operator/cmd/support_archive"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/functional"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
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

const (
	diagExecutorLogFile = "var/lib/dynatrace/remotepluginmodule/log/extensions/diagnostics/diag_executor.log"
	lsLogFile           = "ls.txt"
)

type requiredFiles struct {
	t              *testing.T
	ctx            context.Context
	resources      *resources.Resources
	dk             dynakube.DynaKube
	ec             edgeconnect.EdgeConnect
	collectManaged bool
}

func newRequiredFiles(t *testing.T, ctx context.Context, resources *resources.Resources, customResources CustomResources, collectManaged bool) requiredFiles {
	return requiredFiles{
		t:              t,
		ctx:            ctx,
		resources:      resources,
		dk:             customResources.dk,
		ec:             customResources.ec,
		collectManaged: collectManaged,
	}
}

func (r requiredFiles) collectRequiredFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles, support_archive.OperatorVersionFileName)
	requiredFiles = append(requiredFiles, support_archive.TroublshootOutputFileName)
	requiredFiles = append(requiredFiles, support_archive.SupportArchiveOutputFileName)
	requiredFiles = append(requiredFiles, r.getRequiredPodFiles(labels.AppNameLabel, true)...)
	requiredFiles = append(requiredFiles, r.getRequiredPodFiles(labels.AppManagedByLabel, r.collectManaged)...)
	requiredFiles = append(requiredFiles, r.getRequiredPodDiagnosticLogFiles(r.collectManaged)...)
	requiredFiles = append(requiredFiles, r.getRequiredReplicaSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredServiceFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredWorkloadFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredNamespaceFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredDynaKubeFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredEdgeConnectFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredStatefulSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredDaemonSetFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredWebhookConfigurationFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredCRDFiles()...)
	requiredFiles = append(requiredFiles, r.getRequiredConfigMapFiles()...)

	return requiredFiles
}

func (r requiredFiles) getRequiredPodFiles(labelKey string, collectManaged bool) []string {
	pods := pod.List(r.t, r.ctx, r.resources, r.dk.Namespace)
	requiredFiles := make([]string, 0)

	podList := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		label, ok := podItem.Labels[labelKey]

		return ok && label == operator.DeploymentName
	})

	for _, operatorPod := range podList {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/pod/%s%s",
				support_archive.ManifestsDirectoryName,
				operatorPod.Namespace, operatorPod.Name,
				support_archive.ManifestsFileExtension))
		if collectManaged && (labelKey == "app.kubernetes.io/managed-by" || labelKey == "app.kubernetes.io/name") {
			for _, container := range operatorPod.Spec.Containers {
				requiredFiles = append(requiredFiles,
					fmt.Sprintf("%s/%s/%s.log", support_archive.LogsDirectoryName, operatorPod.Name, container.Name))
			}
		}
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredPodDiagnosticLogFiles(collectManaged bool) []string {
	requiredFiles := make([]string, 0)

	if !collectManaged {
		return requiredFiles
	}

	pods := pod.List(r.t, r.ctx, r.resources, r.dk.Namespace)

	podList := functional.Filter(pods.Items, func(podItem corev1.Pod) bool {
		appNamelabel, okAppNamelabel := podItem.Labels[labels.AppNameLabel]
		appManagedByLabel, okAppManagedByLabel := podItem.Labels[labels.AppManagedByLabel]

		return okAppNamelabel && appNamelabel == support_archive.LabelEecPodName && okAppManagedByLabel && appManagedByLabel == operator.DeploymentName
	})

	for _, pod := range podList {
		requiredFiles = append(requiredFiles, support_archive.BuildZipFilePath(pod.Name, diagExecutorLogFile))
		requiredFiles = append(requiredFiles, support_archive.BuildZipFilePath(pod.Name, lsLogFile))
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredReplicaSetFiles() []string {
	replicaSets := replicaset.List(r.t, r.ctx, r.resources, r.dk.Namespace)
	requiredFiles := make([]string, 0)
	for _, replicaSet := range replicaSets.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/replicaset/%s%s",
				support_archive.ManifestsDirectoryName,
				replicaSet.Namespace, replicaSet.Name,
				support_archive.ManifestsFileExtension))
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredStatefulSetFiles() []string {
	statefulSet, err := statefulset.NewQuery(r.ctx, r.resources, client.ObjectKey{
		Namespace: r.dk.Namespace,
		Name:      "dynakube-activegate"}).Get()
	require.NoError(r.t, err)
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/statefulset/%s%s",
			support_archive.ManifestsDirectoryName,
			statefulSet.Namespace, statefulSet.Name,
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredDaemonSetFiles() []string {
	oneagentDaemonSet, err := oneagent.Get(r.ctx, r.resources, r.dk)
	require.NoError(r.t, err)
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/daemonset/%s%s",
			support_archive.ManifestsDirectoryName,
			oneagentDaemonSet.Namespace,
			oneagentDaemonSet.Name,
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredServiceFiles() []string {
	services := service.List(r.t, r.ctx, r.resources, r.dk.Namespace)
	requiredFiles := make([]string, 0)
	for _, requiredService := range services.Items {
		requiredFiles = append(requiredFiles,
			fmt.Sprintf("%s/%s/service/%s%s",
				support_archive.ManifestsDirectoryName,
				requiredService.Namespace,
				requiredService.Name,
				support_archive.ManifestsFileExtension))
	}

	return requiredFiles
}

func (r requiredFiles) getRequiredWorkloadFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"deployment",
			operator.DeploymentName,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"deployment",
			e2ewebhook.DeploymentName,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"daemonset",
			csi.DaemonSetName,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.ec.Namespace,
			"deployment",
			r.ec.Name,
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredNamespaceFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/namespace-%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			r.dk.Namespace,
			support_archive.ManifestsFileExtension))
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/namespace-%s%s",
			support_archive.ManifestsDirectoryName,
			support_archive.InjectedNamespacesManifestsDirectoryName,
			testAppNameInjected,
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredDynaKubeFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"dynakube",
			r.dk.Name,
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredEdgeConnectFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.ec.Namespace,
			"edgeconnect",
			r.ec.Name,
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredWebhookConfigurationFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			support_archive.WebhookConfigurationsDirectoryName,
			strings.ToLower(support_archive.MutatingWebhookConfigurationKind),
			support_archive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			support_archive.WebhookConfigurationsDirectoryName,
			strings.ToLower(support_archive.ValidatingWebhookConfigurationKind),
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredCRDFiles() []string {
	requiredFiles := make([]string, 0)
	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			support_archive.CRDDirectoryName,
			strings.Join([]string{strings.ToLower(support_archive.CRDKindName), "dynakubes"}, "-"),
			support_archive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			support_archive.CRDDirectoryName,
			strings.Join([]string{strings.ToLower(support_archive.CRDKindName), "edgeconnects"}, "-"),
			support_archive.ManifestsFileExtension))

	return requiredFiles
}

func (r requiredFiles) getRequiredConfigMapFiles() []string {
	requiredFiles := make([]string, 0)

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			"dynatrace-node-cache",
			support_archive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			"kube-root-ca.crt",
			support_archive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s-%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			r.dk.Name,
			"deployment-metadata",
			support_archive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s-%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			r.dk.Name,
			"oneagent-connection-info",
			support_archive.ManifestsFileExtension))

	requiredFiles = append(requiredFiles,
		fmt.Sprintf("%s/%s/%s/%s-%s%s",
			support_archive.ManifestsDirectoryName,
			r.dk.Namespace,
			"configmap",
			r.dk.Name,
			"activegate-connection-info",
			support_archive.ManifestsFileExtension))

	return requiredFiles
}
