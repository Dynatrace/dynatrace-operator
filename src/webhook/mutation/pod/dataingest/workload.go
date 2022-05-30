package dataingest_mutation

import (
	"context"

	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type workloadInfo struct {
	name string
	kind string
}

func (mutator *DataIngestPodMutator) retrieveWorkload(request *dtwebhook.MutationRequest) (*workloadInfo, error) {
	workload, err := findRootOwnerOfPod(request.Context, mutator.metaClient, request.Pod, request.Namespace.Name)
	if err != nil {
		return nil, err
	}
	return workload, nil
}

func findRootOwnerOfPod(ctx context.Context, clt client.Client, pod *corev1.Pod, namespace string) (*workloadInfo, error) {
	obj := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pod.APIVersion,
			Kind:       pod.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pod.ObjectMeta.Name,
			// pod.ObjectMeta.Namespace is empty yet
			Namespace:       namespace,
			OwnerReferences: pod.ObjectMeta.OwnerReferences,
		},
	}
	workloadInfo, err := findRootOwner(ctx, clt, obj)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &workloadInfo, nil
}

func findRootOwner(ctx context.Context, clt client.Client, partialObjectMetadata *metav1.PartialObjectMetadata) (workloadInfo, error) {
	if len(partialObjectMetadata.ObjectMeta.OwnerReferences) == 0 {
		kind := partialObjectMetadata.Kind
		if kind == "Pod" {
			kind = ""
		}
		return workloadInfo{
			name: partialObjectMetadata.ObjectMeta.Name,
			kind: kind,
		}, nil
	}

	objectMetadata := partialObjectMetadata.ObjectMeta
	for _, owner := range objectMetadata.OwnerReferences {
		if owner.Controller != nil && *owner.Controller && isWellKnownWorkload(owner) {
			obj := &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
				},
			}
			if err := clt.Get(ctx, client.ObjectKey{Name: owner.Name, Namespace: objectMetadata.Namespace}, obj); err != nil {
				log.Error(err, "failed to query the object", "apiVersion", owner.APIVersion, "kind", owner.Kind, "name", owner.Name, "namespace", objectMetadata.Namespace)
				return workloadInfo{
					name: partialObjectMetadata.ObjectMeta.Name,
					kind: partialObjectMetadata.Kind,
				}, err
			}

			return findRootOwner(ctx, clt, obj)
		}
	}
	return workloadInfo{
		name: partialObjectMetadata.ObjectMeta.Name,
		kind: partialObjectMetadata.Kind,
	}, nil
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
