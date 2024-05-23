package metadata

import (
	"context"

	kubeobjects "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type workloadInfo struct {
	name string
	kind string
}

func newWorkloadInfo(partialObjectMetadata *metav1.PartialObjectMetadata) workloadInfo {
	return workloadInfo{
		name: partialObjectMetadata.ObjectMeta.Name,
		kind: partialObjectMetadata.Kind,
	}
}

func (mut *Mutator) retrieveWorkload(request *dtwebhook.MutationRequest) (*workloadInfo, error) {
	workload, err := findRootOwnerOfPod(request.Context, mut.metaClient, request.Pod, request.Namespace.Name)
	if err != nil {
		return nil, err
	}

	return workload, nil
}

func findRootOwnerOfPod(ctx context.Context, clt client.Client, pod *corev1.Pod, namespace string) (*workloadInfo, error) {
	podPartialMetadata := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pod.APIVersion,
			Kind:       pod.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeobjects.GetName(*pod),
			// pod.ObjectMeta.Namespace is empty yet
			Namespace:       namespace,
			OwnerReferences: pod.ObjectMeta.OwnerReferences,
		},
	}

	workloadInfo, err := findRootOwner(ctx, clt, podPartialMetadata, podPartialMetadata) // default owner of the pod is the pod itself
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &workloadInfo, nil
}

func findRootOwner(ctx context.Context, clt client.Client, rootObject *metav1.PartialObjectMetadata, wellKnownOwner *metav1.PartialObjectMetadata) (workloadInfo, error) {
	if len(rootObject.ObjectMeta.OwnerReferences) == 0 {
		return newWorkloadInfo(wellKnownOwner), nil
	}

	objectMetadata := rootObject.ObjectMeta
	for _, owner := range objectMetadata.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			ownerObjectMetadata := &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
				},
			}

			err := clt.Get(ctx, client.ObjectKey{Name: owner.Name, Namespace: objectMetadata.Namespace}, ownerObjectMetadata)
			if err != nil {
				log.Error(err, "failed to query the object",
					"apiVersion", owner.APIVersion,
					"kind", owner.Kind,
					"name", owner.Name,
					"namespace", objectMetadata.Namespace,
				)

				return newWorkloadInfo(wellKnownOwner), err
			}

			if isWellKnownWorkload(owner) {
				wellKnownOwner = ownerObjectMetadata
			}

			return findRootOwner(ctx, clt, ownerObjectMetadata, wellKnownOwner)
		}
	}

	return newWorkloadInfo(wellKnownOwner), nil
}

func isWellKnownWorkload(ownerRef metav1.OwnerReference) bool {
	knownWorkloads := []metav1.TypeMeta{
		{Kind: "ReplicaSet", APIVersion: "apps/v1"},
		{Kind: "Deployment", APIVersion: "apps/v1"},
		{Kind: "ReplicationController", APIVersion: "v1"},
		{Kind: "StatefulSet", APIVersion: "apps/v1"},
		{Kind: "DaemonSet", APIVersion: "apps/v1"},
		{Kind: "Job", APIVersion: "batch/v1"},
		{Kind: "CronJob", APIVersion: "batch/v1"},
		{Kind: "DeploymentConfig", APIVersion: "apps.openshift.io/v1"},
	}

	for _, knownController := range knownWorkloads {
		if ownerRef.Kind == knownController.Kind &&
			ownerRef.APIVersion == knownController.APIVersion {
			return true
		}
	}

	return false
}
