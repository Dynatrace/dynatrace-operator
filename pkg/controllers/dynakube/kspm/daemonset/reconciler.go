package daemonset

import (
	"context"
	"maps"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
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
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.KSPM().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		defer meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		query := k8sdaemonset.Query(r.client, r.apiReader, log)

		err := query.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: r.dk.KSPM().GetDaemonSetName(), Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up KSPM daemonset")
		}

		return nil // clean-up shouldn't cause a failure
	}

	ds, err := r.generateDaemonSet()
	if err != nil {
		return err
	}

	updated, err := k8sdaemonset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, ds)
	if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), conditionType, err)

		return err
	}

	if updated {
		conditions.SetDaemonSetOutdated(r.dk.Conditions(), conditionType, r.dk.KSPM().GetDaemonSetName()) // needed to reset the timestamp
		conditions.SetDaemonSetCreated(r.dk.Conditions(), conditionType, r.dk.KSPM().GetDaemonSetName())
	}

	return nil
}

func (r *Reconciler) generateDaemonSet() (*appsv1.DaemonSet, error) {
	tenantUUID, err := r.dk.TenantUUID()
	if err != nil {
		return nil, err
	}

	labels := k8slabel.NewCoreLabels(r.dk.Name, k8slabel.KSPMComponentLabel)
	templateAnnotations := map[string]string{tokenSecretHashAnnotation: r.dk.KSPM().TokenSecretHash}
	maps.Copy(templateAnnotations, r.dk.KSPM().Annotations)

	affinity := k8saffinity.NewAMDOnlyNodeAffinity()
	if r.dk.KSPM().NodeAffinity != nil {
		affinity.NodeAffinity = r.dk.KSPM().NodeAffinity
	}

	ds, err := k8sdaemonset.Build(r.dk, r.dk.KSPM().GetDaemonSetName(), getContainer(*r.dk, tenantUUID),
		k8sdaemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), r.dk.KSPM().Labels),
		k8sdaemonset.SetAllAnnotations(r.dk.KSPM().Annotations, templateAnnotations),
		k8sdaemonset.SetServiceAccount(serviceAccountName),
		k8sdaemonset.SetAffinity(affinity),
		k8sdaemonset.SetPriorityClass(r.dk.KSPM().PriorityClassName),
		k8sdaemonset.SetNodeSelector(r.dk.KSPM().NodeSelector),
		k8sdaemonset.SetTolerations(r.dk.KSPM().Tolerations),
		k8sdaemonset.SetPullSecret(r.dk.ImagePullSecretReferences()...),
		k8sdaemonset.SetUpdateStrategy(r.getUpdateStrategy()),
		k8sdaemonset.SetVolumes(getVolumes(*r.dk)),
		k8sdaemonset.SetAutomountServiceAccountToken(false),
		k8sdaemonset.SetHostPID(true),
	)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func (r *Reconciler) getUpdateStrategy() appsv1.DaemonSetUpdateStrategy {
	updateStrategy := r.dk.KSPM().UpdateStrategy

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
