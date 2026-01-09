package databases

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdeployment"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader

	dk *dynakube.DynaKube
}

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	log.Debug("reconciling deployments")

	query := k8sdeployment.Query(r.client, r.apiReader, log)
	ext := r.dk.Extensions()
	expectedDeploymentNames := make([]string, len(ext.Databases))

	for i, dbSpec := range ext.Databases {
		expectedDeploymentNames[i] = ext.GetDatabaseDatasourceName(dbSpec.ID)
	}

	if err := deleteDeployments(ctx, r.client, r.dk, expectedDeploymentNames); err != nil {
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	for i, dbSpec := range ext.Databases {
		replicas, err := r.getReplicas(ctx, expectedDeploymentNames[i], dbSpec.Replicas)
		if err != nil {
			k8sconditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

			return err
		}

		deploy, err := k8sdeployment.Build(
			r.dk, ext.GetDatabaseDatasourceName(dbSpec.ID),
			k8sdeployment.SetReplicas(replicas),
			k8sdeployment.SetAllLabels(buildAllLabels(r.dk, dbSpec)),
			k8sdeployment.SetAllAnnotations(nil, dbSpec.Annotations),
			k8sdeployment.SetAffinity(dbSpec.Affinity),
			k8sdeployment.SetTolerations(r.dk.Spec.Templates.SQLExtensionExecutor.Tolerations),
			k8sdeployment.SetTopologySpreadConstraints(dbSpec.TopologySpreadConstraints),
			k8sdeployment.SetNodeSelector(dbSpec.NodeSelector),
			k8sdeployment.SetImagePullSecrets(r.dk.ImagePullSecretReferences()),
			k8sdeployment.SetServiceAccount(buildServiceAccountName(dbSpec)),
			k8sdeployment.SetSecurityContext(buildPodSecurityContext()),
			k8sdeployment.SetContainer(buildContainer(r.dk, dbSpec)),
			k8sdeployment.SetVolumes(buildVolumes(r.dk, dbSpec)),
		)
		if err != nil {
			// This error indicates that the scheme is missing required types and is unrecoverable.
			k8sconditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

			return err
		}

		changed, err := query.WithOwner(r.dk).CreateOrUpdate(ctx, deploy)
		if err != nil {
			k8sconditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

			return err
		}

		if changed {
			log.Info("deployment created or updated", "name", deploy.Name)
		}
	}

	if len(expectedDeploymentNames) > 0 {
		k8sconditions.SetDeploymentsApplied(r.dk, conditionType, expectedDeploymentNames)
	} else {
		_ = meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)
	}

	return nil
}

// To work well with horizontal pod autoscalers, ensure that we use external changes to replicas and not overwrite it.
func (r *Reconciler) getReplicas(ctx context.Context, name string, defaultReplicas *int32) (int32, error) {
	if defaultReplicas != nil {
		return *defaultReplicas, nil
	}

	deploy, err := k8sdeployment.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKey{Namespace: r.dk.Namespace, Name: name})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return 1, nil
		}

		return 0, err
	}

	return *deploy.Spec.Replicas, nil
}
