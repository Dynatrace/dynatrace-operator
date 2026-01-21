package daemonset

import (
	"context"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdaemonset"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceAccountName = "dynatrace-node-config-collector"
)

type Reconciler struct {
	daemonset k8sdaemonset.QueryObject
}

func NewReconciler(clt client.Client, apiReader client.Reader) *Reconciler {
	return &Reconciler{
		daemonset: k8sdaemonset.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	if !dk.KSPM().IsEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), conditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		defer meta.RemoveStatusCondition(dk.Conditions(), conditionType)

		err := r.daemonset.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dk.KSPM().GetDaemonSetName(), Namespace: dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up KSPM daemonset")
		}

		return nil // clean-up shouldn't cause a failure
	}

	ds, err := r.generateDaemonSet(dk)
	if err != nil {
		return err
	}

	updated, err := r.daemonset.WithOwner(dk).CreateOrUpdate(ctx, ds)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), conditionType, err)

		return err
	}

	if updated {
		k8sconditions.SetDaemonSetOutdated(dk.Conditions(), conditionType, dk.KSPM().GetDaemonSetName()) // needed to reset the timestamp
		k8sconditions.SetDaemonSetCreated(dk.Conditions(), conditionType, dk.KSPM().GetDaemonSetName())
	}

	return nil
}

func (r *Reconciler) generateDaemonSet(dk *dynakube.DynaKube) (*appsv1.DaemonSet, error) {
	tenantUUID, err := dk.TenantUUID()
	if err != nil {
		return nil, err
	}

	labels := k8slabel.NewCoreLabels(dk.Name, k8slabel.KSPMComponentLabel)
	templateAnnotations := map[string]string{tokenSecretHashAnnotation: dk.KSPM().TokenSecretHash}
	maps.Copy(templateAnnotations, dk.KSPM().Annotations)

	affinity := k8saffinity.NewAMDOnlyNodeAffinity()
	if dk.KSPM().NodeAffinity != nil {
		affinity.NodeAffinity = dk.KSPM().NodeAffinity
	}

	ds, err := k8sdaemonset.Build(dk, dk.KSPM().GetDaemonSetName(), getContainer(*dk, tenantUUID),
		k8sdaemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), dk.KSPM().Labels),
		k8sdaemonset.SetAllAnnotations(dk.KSPM().Annotations, templateAnnotations),
		k8sdaemonset.SetServiceAccount(serviceAccountName),
		k8sdaemonset.SetAffinity(affinity),
		k8sdaemonset.SetPriorityClass(dk.KSPM().PriorityClassName),
		k8sdaemonset.SetNodeSelector(dk.KSPM().NodeSelector),
		k8sdaemonset.SetTolerations(dk.KSPM().Tolerations),
		k8sdaemonset.SetPullSecret(dk.ImagePullSecretReferences()...),
		k8sdaemonset.SetUpdateStrategy(r.getUpdateStrategy(dk)),
		k8sdaemonset.SetVolumes(getVolumes(*dk)),
		k8sdaemonset.SetAutomountServiceAccountToken(false),
		k8sdaemonset.SetHostPID(true),
	)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func (r *Reconciler) getUpdateStrategy(dk *dynakube.DynaKube) appsv1.DaemonSetUpdateStrategy {
	updateStrategy := dk.KSPM().UpdateStrategy

	if updateStrategy != nil {
		return *updateStrategy
	}

	return appsv1.DaemonSetUpdateStrategy{
		Type: appsv1.RollingUpdateDaemonSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDaemonSet{
			MaxUnavailable: getDefaultMaxUnavailable(),
			MaxSurge:       getDefaultMaxSurge(),
		},
	}
}

func getDefaultMaxUnavailable() *intstr.IntOrString {
	defaultMaxUnavailable := "25%"

	return &intstr.IntOrString{StrVal: defaultMaxUnavailable}
}

func getDefaultMaxSurge() *intstr.IntOrString {
	defaultMaxSurge := 1

	return &intstr.IntOrString{IntVal: int32(defaultMaxSurge)}
}
