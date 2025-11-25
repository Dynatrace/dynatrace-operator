package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8saffinity"
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

var KubernetesSettingsNotAvailableError = errors.New("the status of the DynaKube is missing information about the kubernetes monitored-entity, skipping LogMonitoring deployment until it is ready")

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.LogMonitoring().IsStandalone() {
		if meta.FindStatusCondition(*r.dk.Conditions(), ConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := k8sdaemonset.Query(r.client, r.apiReader, log)

		err := query.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: r.dk.LogMonitoring().GetDaemonSetName(), Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up LogMonitoring config-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), ConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	ds, err := r.generateDaemonSet()
	if err != nil {
		return err
	}

	updated, err := k8sdaemonset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, ds)
	if err != nil {
		conditions.SetKubeAPIError(r.dk.Conditions(), ConditionType, err)

		return err
	}

	if updated {
		conditions.SetDaemonSetOutdated(r.dk.Conditions(), ConditionType, r.dk.LogMonitoring().GetDaemonSetName()) // needed to reset the timestamp
		conditions.SetDaemonSetCreated(r.dk.Conditions(), ConditionType, r.dk.LogMonitoring().GetDaemonSetName())
	}

	return nil
}

func (r *Reconciler) generateDaemonSet() (*appsv1.DaemonSet, error) {
	tenantUUID, err := r.dk.TenantUUID()
	if err != nil {
		return nil, err
	}

	labels := k8slabel.NewCoreLabels(r.dk.Name, k8slabel.LogMonitoringComponentLabel)

	maxUnavailable := intstr.FromInt(r.dk.FF().GetOneAgentMaxUnavailable())

	ds, err := k8sdaemonset.Build(r.dk, r.dk.LogMonitoring().GetDaemonSetName(), getContainer(*r.dk),
		k8sdaemonset.SetInitContainer(getInitContainer(*r.dk)),
		k8sdaemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), r.dk.LogMonitoring().Template().Labels),
		k8sdaemonset.SetAllAnnotations(nil, r.getAnnotations()),
		k8sdaemonset.SetServiceAccount(serviceAccountName),
		k8sdaemonset.SetDNSPolicy(r.dk.LogMonitoring().Template().DNSPolicy),
		k8sdaemonset.SetAffinity(k8saffinity.NewMultiArchNodeAffinity()),
		k8sdaemonset.SetPriorityClass(r.dk.LogMonitoring().Template().PriorityClassName),
		k8sdaemonset.SetNodeSelector(r.dk.LogMonitoring().Template().NodeSelector),
		k8sdaemonset.SetTolerations(r.dk.LogMonitoring().Template().Tolerations),
		k8sdaemonset.SetPullSecret(r.dk.ImagePullSecretReferences()...),
		k8sdaemonset.SetUpdateStrategy(appsv1.DaemonSetUpdateStrategy{
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &maxUnavailable,
			},
		}),
		k8sdaemonset.SetVolumes(getVolumes(r.dk.Name, tenantUUID)),
	)
	if err != nil {
		return nil, err
	}

	return ds, nil
}

func isMEConfigured(dk dynakube.DynaKube) bool {
	return dk.Status.KubernetesClusterMEID != "" && dk.Status.KubernetesClusterName != ""
}
