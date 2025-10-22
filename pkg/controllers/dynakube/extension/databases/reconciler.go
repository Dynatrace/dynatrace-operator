package databases

import (
	"context"
	"errors"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/deployment"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynaKube condition that is managed by the reconciler.
const conditionType = "DatabaseDatasourcesAvailable"

var log = logd.Get().WithName("extension-databases")

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

	if cond := meta.FindStatusCondition(r.dk.Status.Conditions, conditionType); cond != nil &&
		cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == r.dk.Generation {
		return nil
	}

	query := deployment.Query(r.client, r.apiReader, log)
	ext := r.dk.Extensions()
	expectedDeploymentNames := make([]string, len(ext.Databases))

	for i, dbSpec := range ext.Databases {
		expectedDeploymentNames[i] = ext.GetDatabaseDatasourceName(dbSpec.ID)
	}

	if err := deleteDeployments(ctx, r.client, r.dk, expectedDeploymentNames); err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	var buildErrors error

	for i, dbSpec := range ext.Databases {
		replicas, err := r.getReplicas(ctx, expectedDeploymentNames[i], dbSpec.Replicas)
		if err != nil {
			conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

			return err
		}

		deploy, err := deployment.Build(
			r.dk, ext.GetDatabaseDatasourceName(dbSpec.ID),
			deployment.SetReplicas(replicas),
			deployment.SetAllLabels(buildAllLabels(r.dk, dbSpec)),
			deployment.SetAllAnnotations(nil, dbSpec.Annotations),
			deployment.SetAffinity(dbSpec.Affinity),
			deployment.SetTolerations(r.dk.Spec.Templates.DatabaseExecutor.Tolerations),
			deployment.SetTopologySpreadConstraints(dbSpec.TopologySpreadConstraints),
			deployment.SetNodeSelector(dbSpec.NodeSelector),
			deployment.SetImagePullSecrets(r.dk.ImagePullSecretReferences()),
			deployment.SetServiceAccount(buildServiceAccountName(dbSpec)),
			deployment.SetSecurityContext(buildPodSecurityContext()),
			deployment.SetContainer(buildContainer(r.dk, dbSpec)),
			deployment.SetVolumes(buildVolumes(r.dk, dbSpec)),
		)
		if err != nil {
			// Not a critical error. Next deployment could succeed.
			buildErrors = errors.Join(buildErrors, err)

			continue
		}

		changed, err := query.WithOwner(r.dk).CreateOrUpdate(ctx, deploy)
		if err != nil {
			conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

			// Surface previous errors if there are any
			return errors.Join(err, buildErrors)
		}

		if changed {
			log.Info("deployment created or updated", "name", deploy.Name)
		}
	}

	if buildErrors != nil {
		return buildErrors
	}

	if len(expectedDeploymentNames) > 0 {
		conditions.SetDeploymentsApplied(r.dk, conditionType, expectedDeploymentNames)
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

	deploy, err := deployment.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKey{Namespace: r.dk.Namespace, Name: name})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return 1, nil
		}

		return 0, err
	}

	return *deploy.Spec.Replicas, nil
}
