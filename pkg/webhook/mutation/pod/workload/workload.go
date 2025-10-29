package workload

import (
	"context"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	kubeobjects "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/pod"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Info struct {
	Name string
	Kind string
}

func NewInfo(partialObjectMetadata *metav1.PartialObjectMetadata) *Info {
	return &Info{
		Name: partialObjectMetadata.Name,

		// workload kind in lower case according to dt semantic-dictionary
		// https://docs.dynatrace.com/docs/discover-dynatrace/references/semantic-dictionary/fields#kubernetes
		Kind: strings.ToLower(partialObjectMetadata.Kind),
	}
}

func FindRootOwnerOfPod(ctx context.Context, clt client.Client, request *dtwebhook.BaseRequest, log logd.Logger) (*Info, error) {
	podPartialMetadata := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: request.Pod.APIVersion,
			Kind:       request.Pod.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: kubeobjects.GetName(*request.Pod),
			// pod.Namespace is empty yet
			Namespace:       request.Namespace.Name,
			OwnerReferences: request.Pod.OwnerReferences,
		},
	}

	rootOwner, err := findRootOwner(ctx, clt, podPartialMetadata, log) // default owner of the pod is the pod itself
	if err != nil {
		return nil, err
	}

	return NewInfo(rootOwner), nil
}

func findRootOwner(ctx context.Context, clt client.Client, childObjectMetadata *metav1.PartialObjectMetadata, log logd.Logger) (parentObjectMetadata *metav1.PartialObjectMetadata, err error) {
	objectMetadata := childObjectMetadata.ObjectMeta
	for _, owner := range objectMetadata.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			parentObjectMetadata = &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
				},
			}

			if !isWellKnownWorkload(parentObjectMetadata) {
				return childObjectMetadata, nil
			}

			err = clt.Get(ctx, client.ObjectKey{Name: owner.Name, Namespace: objectMetadata.Namespace}, parentObjectMetadata)
			if err != nil {
				log.Error(err, "failed to query the object",
					"apiVersion", owner.APIVersion,
					"kind", owner.Kind,
					"name", owner.Name,
					"namespace", objectMetadata.Namespace,
				)

				return childObjectMetadata, err
			}

			parentObjectMetadata, err = findRootOwner(ctx, clt, parentObjectMetadata, log)
			if err != nil {
				return childObjectMetadata, err
			}

			return parentObjectMetadata, nil
		}
	}

	return childObjectMetadata, nil
}

func isWellKnownWorkload(ownerRef *metav1.PartialObjectMetadata) bool {
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
