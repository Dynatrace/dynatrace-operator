package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sdaemonset"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceAccountName = "dynatrace-logmonitoring"
)

type Reconciler struct {
	daemonset k8sdaemonset.QueryObject
}

func NewReconciler(clt client.Client,
	apiReader client.Reader) *Reconciler {
	return &Reconciler{
		daemonset: k8sdaemonset.Query(clt, apiReader, log),
	}
}

var KubernetesSettingsNotAvailableError = errors.New("the status of the DynaKube is missing information about the kubernetes monitored-entity, skipping LogMonitoring deployment until it is ready")

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	if !dk.LogMonitoring().IsStandalone() {
		if meta.FindStatusCondition(*dk.Conditions(), ConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		err := r.daemonset.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: dk.LogMonitoring().GetDaemonSetName(), Namespace: dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up LogMonitoring config-secret")
		}

		meta.RemoveStatusCondition(dk.Conditions(), ConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	ds, err := r.generateDaemonSet(dk)
	if err != nil {
		return err
	}

	updated, err := r.daemonset.WithOwner(dk).CreateOrUpdate(ctx, ds)
	if err != nil {
		k8sconditions.SetKubeAPIError(dk.Conditions(), ConditionType, err)

		return err
	}

	if updated {
		k8sconditions.SetDaemonSetOutdated(dk.Conditions(), ConditionType, dk.LogMonitoring().GetDaemonSetName()) // needed to reset the timestamp
		k8sconditions.SetDaemonSetCreated(dk.Conditions(), ConditionType, dk.LogMonitoring().GetDaemonSetName())
	}

	return nil
}

func (r *Reconciler) generateDaemonSet(dk *dynakube.DynaKube) (*appsv1.DaemonSet, error) {
	tenantUUID, err := dk.TenantUUID()
	if err != nil {
		return nil, err
	}

	labels := k8slabel.NewCoreLabels(dk.Name, k8slabel.LogMonitoringComponentLabel)

	ds, err := k8sdaemonset.Build(dk, dk.LogMonitoring().GetDaemonSetName(), getContainer(*dk, tenantUUID),
		k8sdaemonset.SetInitContainer(getInitContainer(*dk, tenantUUID)),
		k8sdaemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), dk.LogMonitoring().Template().Labels),
		k8sdaemonset.SetAllAnnotations(nil, r.getAnnotations(dk)),
		k8sdaemonset.SetServiceAccount(serviceAccountName),
		k8sdaemonset.SetDNSPolicy(dk.LogMonitoring().Template().DNSPolicy),
		k8sdaemonset.SetAffinity(k8saffinity.NewMultiArchNodeAffinity()),
		k8sdaemonset.SetPriorityClass(dk.LogMonitoring().Template().PriorityClassName),
		k8sdaemonset.SetNodeSelector(dk.LogMonitoring().Template().NodeSelector),
		k8sdaemonset.SetTolerations(dk.LogMonitoring().Template().Tolerations),
		k8sdaemonset.SetPullSecret(dk.ImagePullSecretReferences()...),
		k8sdaemonset.SetUpdateStrategy(r.getUpdateStrategy(dk)),
		k8sdaemonset.SetVolumes(getVolumes(dk.Name)),
	)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func (r *Reconciler) getUpdateStrategy(dk *dynakube.DynaKube) appsv1.DaemonSetUpdateStrategy {
	maxUnavailable := intstr.FromInt(dk.FF().GetOneAgentMaxUnavailable()) //nolint:staticcheck

	us := appsv1.DaemonSetUpdateStrategy{
		RollingUpdate: &appsv1.RollingUpdateDaemonSet{
			MaxUnavailable: &maxUnavailable,
		},
	}

	if dk.LogMonitoring().Template().RollingUpdate != nil {
		us.RollingUpdate = dk.LogMonitoring().Template().RollingUpdate
	}

	return us
}

func isMEConfigured(dk dynakube.DynaKube) bool {
	return dk.Status.KubernetesClusterMEID != "" && dk.Status.KubernetesClusterName != ""
}
