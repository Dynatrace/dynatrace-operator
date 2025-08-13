package daemonset

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
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

		query := daemonset.Query(r.client, r.apiReader, log)

		err := query.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: r.dk.LogMonitoring().GetDaemonSetName(), Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up LogMonitoring config-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), ConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	hasAllScopes := r.hasReadScope() && r.hasWriteScope()

	if hasAllScopes && !r.isMEConfigured() {
		log.Info("Kubernetes settings are not yet available and both settings.write and settings.read token scopes are availaible, will requeue")

		return KubernetesSettingsNotAvailableError
	}

	useMetadata := r.isMEConfigured() && hasAllScopes

	ds, err := r.generateDaemonSet(useMetadata)
	if err != nil {
		return err
	}

	updated, err := daemonset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, ds)
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

func (r *Reconciler) generateDaemonSet(useMetadata bool) (*appsv1.DaemonSet, error) {
	tenantUUID, err := r.dk.TenantUUID()
	if err != nil {
		return nil, err
	}

	labels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.LogMonitoringComponentLabel)

	maxUnavailable := intstr.FromInt(r.dk.FF().GetOneAgentMaxUnavailable())

	ds, err := daemonset.Build(r.dk, r.dk.LogMonitoring().GetDaemonSetName(), getContainer(*r.dk, tenantUUID),
		daemonset.SetInitContainer(getInitContainer(*r.dk, tenantUUID, useMetadata)),
		daemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), r.dk.LogMonitoring().Template().Labels),
		daemonset.SetAllAnnotations(nil, r.getAnnotations()),
		daemonset.SetServiceAccount(serviceAccountName),
		daemonset.SetDNSPolicy(r.dk.LogMonitoring().Template().DNSPolicy),
		daemonset.SetAffinity(node.Affinity()),
		daemonset.SetPriorityClass(r.dk.LogMonitoring().Template().PriorityClassName),
		daemonset.SetNodeSelector(r.dk.LogMonitoring().Template().NodeSelector),
		daemonset.SetTolerations(r.dk.LogMonitoring().Template().Tolerations),
		daemonset.SetPullSecret(r.dk.ImagePullSecretReferences()...),
		daemonset.SetUpdateStrategy(appsv1.DaemonSetUpdateStrategy{
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &maxUnavailable,
			},
		}),
		daemonset.SetVolumes(getVolumes(r.dk.Name)),
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

func (r *Reconciler) hasReadScope() bool {
	return conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsRead)
}

func (r *Reconciler) hasWriteScope() bool {
	return conditions.IsOptionalScopeAvailable(r.dk, dtclient.ConditionTypeAPITokenSettingsWrite)
}

func (r *Reconciler) isMEConfigured() bool {
	return r.dk.Status.KubernetesClusterMEID != "" && r.dk.Status.KubernetesClusterName != ""
}
