package otel

import (
	"context"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/hash"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	appsv1 "k8s.io/api/apps/v1"
)

const (
	statefulsetName = "dynatrace-extensions-collector"
)

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {

	sts, err := statefulset.Build(r.dk, statefulsetName, buildContainer(r.dk),
		statefulset.SetReplicas(1),
		statefulset.SetPodManagementPolicy(appsv1.ParallelPodManagement),
		statefulset.SetAllLabels(appLabels.BuildLabels(), appLabels.BuildMatchLabels(), appLabels.BuildLabels(), r.dk.Spec.Templates.ExtensionExecutionController.Labels),
		statefulset.SetAllAnnotations(nil, r.dk.Spec.Templates.ExtensionExecutionController.Annotations),
		statefulset.SetAffinity(buildAffinity()),
		statefulset.SetTolerations(r.dk.Spec.Templates.ExtensionExecutionController.Tolerations),
		statefulset.SetTopologySpreadConstraints(buildTopologySpreadConstraints(r.dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints, r.dk.Name)),
		statefulset.SetServiceAccount(serviceAccountName),
		statefulset.SetSecurityContext(buildPodSecurityContext()),
		statefulset.SetUpdateStrategy(buildUpdateStrategy()),
		setTlsRef(r.dk.Spec.Templates.ExtensionExecutionController.TlsRefName),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk.Name, r.dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim),
	)

	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), otelControllerStatefulSetConditionType, err)

		return err
	}

	if err := hash.SetHash(sts); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), otelControllerStatefulSetConditionType, err)

		return err
	}

	_, err = statefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, sts)
	if err != nil {
		log.Info("failed to create/update " + statefulsetName + " statefulset")
		conditions.SetKubeApiError(r.dk.Conditions(), otelControllerStatefulSetConditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), otelControllerStatefulSetConditionType, sts.Name)

	return nil
}
