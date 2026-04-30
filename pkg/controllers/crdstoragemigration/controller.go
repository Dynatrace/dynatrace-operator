package crdstoragemigration

import (
	"context"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/webhook"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const retryInterval = 10 * time.Second

// for unit tests
var run = Run

func InitReconcile(ctx context.Context, clt client.Client, namespace string) error {
	ctx, log := logd.NewFromContext(ctx, "crd-storage-migration")
	request := types.NamespacedName{Name: webhook.DeploymentName, Namespace: namespace}

	return wait.PollUntilContextCancel(ctx, retryInterval, true, func(ctx context.Context) (bool, error) {
		log.Info("reconciling CRD storage version migration", "namespace", request.Namespace, "name", request.Name)

		deploy := &appsv1.Deployment{}
		if err := clt.Get(ctx, request, deploy); err != nil {
			if k8serrors.IsNotFound(err) {
				log.Info("no webhook deployment found, skipping CRD storage version migration")

				return true, nil
			}

			log.Error(err, "failed webhook deployment lookup")

			return false, nil
		}

		if !isDeploymentReady(deploy) {
			log.Info("webhook deployment not ready yet, retrying CRD storage version migration later")

			return false, nil
		}

		if err := run(ctx, clt, request.Namespace); err != nil {
			log.Error(err, "CRD storage migration failed")

			return false, nil
		}

		return true, nil
	})
}

func isDeploymentReady(deploy *appsv1.Deployment) bool {
	replicas := ptr.Deref(deploy.Spec.Replicas, 1)

	return deploy.Generation == deploy.Status.ObservedGeneration && replicas > 0 && replicas == deploy.Status.ReadyReplicas
}
