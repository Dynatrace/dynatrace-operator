package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/daemonset"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	nameSuffix         = "-logmonitoring"
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

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.LogMonitoring().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := daemonset.Query(r.client, r.apiReader, log)
		err := query.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: r.dk.LogMonitoring().GetDaemonSetName(), Namespace: r.dk.Namespace}})

		if err != nil {
			log.Error(err, "failed to clean-up LogMonitoring config-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		return nil // clean-up shouldn't cause a failure
	}

	if !r.isMEConfigured() {
		return errors.New("the status of the DynaKube is missing information about the kubernetes monitored-entity, skipping logmodule deployment")
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
		conditions.SetDaemonSetOutdated(r.dk.Conditions(), conditionType, r.dk.LogMonitoring().GetDaemonSetName()) // needed to reset the timestamp
		conditions.SetDaemonSetCreated(r.dk.Conditions(), conditionType, r.dk.LogMonitoring().GetDaemonSetName())
	}

	return nil
}

func (r *Reconciler) generateDaemonSet() (*appsv1.DaemonSet, error) {
	tenantUUID, err := r.dk.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		return nil, err
	}

	labels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.LogMonitoringComponentLabel)

	maxUnavailable := intstr.FromInt(r.dk.FeatureOneAgentMaxUnavailable())

	ds, err := daemonset.Build(r.dk, r.dk.LogMonitoring().GetDaemonSetName(), getContainer(*r.dk),
		daemonset.SetInitContainer(getInitContainer(*r.dk)),
		daemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), r.dk.LogMonitoring().Labels),
		daemonset.SetAllAnnotations(nil, r.dk.LogMonitoring().Annotations),
		daemonset.SetServiceAccount(serviceAccountName),
		daemonset.SetDNSPolicy(r.dk.LogMonitoring().DNSPolicy),
		daemonset.SetAffinity(node.Affinity()),
		daemonset.SetPriorityClass(r.dk.LogMonitoring().PriorityClassName),
		daemonset.SetTolerations(r.dk.LogMonitoring().Tolerations),
		daemonset.SetPullSecret(r.dk.ImagePullSecretReferences()...),
		daemonset.SetUpdateStrategy(appsv1.DaemonSetUpdateStrategy{
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &maxUnavailable,
			},
		}),
		daemonset.SetVolumes(getVolumes(r.dk.Name, tenantUUID)),
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

func (r *Reconciler) isMEConfigured() bool {
	return r.dk.Status.KubernetesClusterMEID != "" && r.dk.Status.KubernetesClusterName != ""
}
