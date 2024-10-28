package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/daemonset"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nameSuffix         = "-node-config-collector"
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

		query := daemonset.Query(r.client, r.apiReader, log)
		err := query.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: r.dk.KSPM().GetDaemonSetName(), Namespace: r.dk.Namespace}})

		if err != nil {
			log.Error(err, "failed to clean-up KSPM config-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		return nil // clean-up shouldn't cause a failure
	}

	ds, err := r.generateDaemonSet()
	if err != nil {
		return err
	}

	updated, err := daemonset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, ds)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)

		return err
	}

	if updated {
		conditions.SetDaemonSetOutdated(r.dk.Conditions(), conditionType, r.dk.KSPM().GetDaemonSetName()) // needed to reset the timestamp
		conditions.SetDaemonSetCreated(r.dk.Conditions(), conditionType, r.dk.KSPM().GetDaemonSetName())
	}

	return nil
}

func (r *Reconciler) generateDaemonSet() (*appsv1.DaemonSet, error) {
	tenantUUID, err := r.dk.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		return nil, err
	}

	labels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.KSPMComponentLabel)

	ds, err := daemonset.Build(r.dk, r.dk.KSPM().GetDaemonSetName(), getContainer(*r.dk, tenantUUID),
		daemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), r.dk.KSPM().Labels),
		daemonset.SetAllAnnotations(map[string]string{tokenSecretHashAnnotation: r.dk.KSPM().TokenSecretHash}, r.dk.KSPM().Annotations),
		daemonset.SetServiceAccount(serviceAccountName),
		daemonset.SetAffinity(node.Affinity()),
		daemonset.SetPriorityClass(r.dk.KSPM().PriorityClassName),
		daemonset.SetTolerations(r.dk.KSPM().Tolerations),
		daemonset.SetPullSecret(r.dk.ImagePullSecretReferences()...),
		daemonset.SetUpdateStrategy(r.getUpdateStrategy()),
		daemonset.SetVolumes(getVolumes(*r.dk)),
		daemonset.SetAutomountServiceAccountToken(false),
		daemonset.SetHostPID(true),
	)

	if err != nil {
		return nil, err
	}

	err = hasher.AddAnnotation(ds)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func (r *Reconciler) getUpdateStrategy() appsv1.DaemonSetUpdateStrategy {
	updateStrategy := r.dk.KSPM().UpdateStrategy

	if updateStrategy.Type == "" {
		updateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType
	}

	if updateStrategy.Type == appsv1.RollingUpdateDaemonSetStrategyType && updateStrategy.RollingUpdate == nil {
		updateStrategy = appsv1.DaemonSetUpdateStrategy{
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: getDefaultMaxUnavailable(),
				MaxSurge:       getDefaultMaxSurge(),
			},
		}
	}

	return updateStrategy
}

func getDefaultMaxUnavailable() *intstr.IntOrString {
	defaultMaxUnavailable := "25%"

	return &intstr.IntOrString{StrVal: defaultMaxUnavailable}
}

func getDefaultMaxSurge() *intstr.IntOrString {
	defaultMaxSurge := 1

	return &intstr.IntOrString{IntVal: int32(defaultMaxSurge)}
}
