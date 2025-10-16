package databases

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/deployment"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
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

	query := deployment.Query(r.client, r.apiReader, log)
	ext := r.dk.Extensions()
	applyNames := make([]string, len(ext.Databases))

	for i, dbex := range ext.Databases {
		applyNames[i] = ext.GetDatabaseExecutorName(dbex.ID)
	}

	if err := deleteDeployments(ctx, r.client, r.dk, applyNames); err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	for i, dbex := range ext.Databases {
		replicas, err := r.getReplicas(ctx, applyNames[i], dbex.Replicas)
		if err != nil {
			conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

			return err
		}

		deploy, err := deployment.Build(
			r.dk, ext.GetDatabaseExecutorName(dbex.ID),
			deployment.SetReplicas(replicas),
			deployment.SetAllLabels(buildAllLabels(r.dk, dbex)),
			deployment.SetAllAnnotations(nil, dbex.Annotations),
			deployment.SetAffinity(dbex.Affinity),
			deployment.SetTolerations(r.dk.Spec.Templates.DatabaseExecutor.Tolerations),
			deployment.SetTopologySpreadConstraints(dbex.TopologySpreadConstraints),
			deployment.SetNodeSelector(dbex.NodeSelector),
			deployment.SetImagePullSecrets(r.dk.ImagePullSecretReferences()),
			deployment.SetServiceAccount(buildServiceAccountName(dbex)),
			deployment.SetSecurityContext(buildPodSecurityContext()),
			deployment.SetContainer(buildContainer(r.dk, dbex)),
			deployment.SetVolumes(buildVolumes(r.dk, dbex)),
		)
		if err != nil {
			return err
		}

		if err := hasher.AddAnnotation(deploy); err != nil {
			return err
		}

		changed, err := query.WithOwner(r.dk).CreateOrUpdate(ctx, deploy)
		if err != nil {
			conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

			return err
		}

		if changed {
			log.Info("ensured deployment", "name", deploy.Name)
		}
	}

	if len(applyNames) > 0 {
		conditions.SetDeploymentsApplied(r.dk.Conditions(), conditionType, applyNames)
	} else {
		_ = meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)
	}

	return nil
}

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
