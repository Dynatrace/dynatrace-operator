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
	nameSuffix         = "-logmodule"
	serviceAccountName = "dynatrace-logmodule"
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

// TODO:
// - Add tests
func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.NeedsLogModule() {
		if meta.FindStatusCondition(*r.dk.Conditions(), conditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := daemonset.Query(r.client, r.apiReader, log)
		err := query.Delete(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: GetName(r.dk.Name), Namespace: r.dk.Namespace}})

		if err != nil {
			log.Error(err, "failed to clean-up LogModule config-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), conditionType)

		return nil // clean-up shouldn't cause a failure
	}

	labels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.LogModuleComponentLabel)

	maxUnavailable := intstr.FromInt(r.dk.FeatureOneAgentMaxUnavailable())

	ds, err := daemonset.Build(r.dk, GetName(r.dk.Name), getContainer(*r.dk),
		daemonset.SetInitContainer(getInitContainer(*r.dk)),
		daemonset.SetAllLabels(labels.BuildLabels(), labels.BuildMatchLabels(), labels.BuildLabels(), r.dk.LogModuleTemplates().Labels),
		daemonset.SetAllAnnotations(nil, r.dk.LogModuleTemplates().Annotations),
		daemonset.SetServiceAccount(serviceAccountName),
		daemonset.SetDNSPolicy(r.dk.LogModuleTemplates().DNSPolicy),
		daemonset.SetAffinity(node.Affinity()),
		daemonset.SetPriorityClass(r.dk.LogModuleTemplates().PriorityClassName),
		daemonset.SetTolerations(r.dk.LogModuleTemplates().Tolerations),
		daemonset.SetPullSecret(r.dk.ImagePullSecretReferences()...),
		daemonset.SetUpdateStrategy(appsv1.DaemonSetUpdateStrategy{
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &maxUnavailable,
			},
		}),
		daemonset.SetVolumes(getVolumes(r.dk.Name)),
	)
	if err != nil {
		return err
	}

	err = hasher.AddAnnotation(ds)
	if err != nil {
		return err
	}

	updated, err := daemonset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, ds)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), conditionType, err)

		return err
	}

	if updated {
		conditions.SetDaemonSetOutdated(r.dk.Conditions(), conditionType, GetName(r.dk.Name)) // needed to reset the timestamp
		conditions.SetDaemonSetCreated(r.dk.Conditions(), conditionType, GetName(r.dk.Name))
	}

	return nil
}

func GetName(dkName string) string {
	return dkName + nameSuffix
}
