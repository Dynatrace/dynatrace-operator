package routing

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/capability"
	"github.com/Dynatrace/dynatrace-operator/controllers/dtversion"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	module            = "msgrouter"
	StatefulSetSuffix = "-" + module
	capabilityName    = "MSGrouter"
	DTDNSEntryPoint   = "DT_DNS_ENTRY_POINT"
)

type Reconciler struct {
	*capability.Reconciler
	log logr.Logger
}

func NewReconciler(clt client.Client, apiReader client.Reader, scheme *runtime.Scheme, dtc dtclient.Client, log logr.Logger,
	instance *v1alpha1.DynaKube, imageVersionProvider dtversion.ImageVersionProvider, enableUpdates bool) *Reconciler {
	baseReconciler := capability.NewReconciler(
		clt, apiReader, scheme, dtc, log, instance, imageVersionProvider, enableUpdates,
		&instance.Spec.RoutingSpec.CapabilityProperties, module, capabilityName, "")
	baseReconciler.AddOnAfterStatefulSetCreate(addDNSEntryPoint(instance))
	return &Reconciler{
		Reconciler: baseReconciler,
		log:        log,
	}
}

func addDNSEntryPoint(instance *v1alpha1.DynaKube) capability.StatefulSetEvent {
	return func(sts *appsv1.StatefulSet) {
		sts.Spec.Template.Spec.Containers[0].Env = append(sts.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  DTDNSEntryPoint,
				Value: buildDNSEntryPoint(instance),
			})
	}
}

func (r *Reconciler) Reconcile() (update bool, err error) {
	update, err = r.Reconciler.Reconcile()
	if update || err != nil {
		return update, errors.WithStack(err)
	}

	return r.createServiceIfNotExists()
}

func (r *Reconciler) createServiceIfNotExists() (bool, error) {
	service := createService(r.Instance, module)
	err := r.Get(context.TODO(), client.ObjectKey{Name: service.Name, Namespace: service.Namespace}, &service)
	if err != nil && k8serrors.IsNotFound(err) {
		r.log.Info("creating service for msgrouter")
		err = r.Create(context.TODO(), &service)
		return true, errors.WithStack(err)
	}
	return false, errors.WithStack(err)
}

// Alternative to event based approach
// Removed before merging routing capability
//
//func (r *Reconciler) setDNSEntryPoint() (bool, error) {
//	sts := &appsv1.StatefulSet{}
//	err := r.Get(context.TODO(), client.ObjectKey{Name: r.buildStatefulSetName(), Namespace: r.Instance.Namespace}, sts)
//	if err != nil {
//		return false, errors.WithStack(err)
//	}
//	if len(sts.Spec.Template.Spec.Containers) <= 0 {
//		return false, errors.New("stateful set for " + module + " is invalid, it has no container")
//	}
//
//	envs := sts.Spec.Template.Spec.Containers[0].Env
//	desiredEnv := corev1.EnvVar{
//						Name:  DTDNSEntryPoint,
//						Value: buildDNSEntryPoint(r.Instance),
//					}
//	for i, env := range envs {
//		if env.Name == desiredEnv.Name {
//			if env.Value == desiredEnv.Value {
//				return false, nil
//			}
//			envs = append(envs[:i], envs[i+1:]...)
//			break
//		}
//	}
//
//	sts.Spec.Template.Spec.Containers[0].Env = append(envs, desiredEnv)
//	err = r.Update(context.TODO(), sts)
//	if err != nil {
//		return false, errors.WithStack(err)
//	}
//	return true, nil
//}
//func (r *Reconciler) buildStatefulSetName() string {
//	return r.Instance.Name + "-" + module
//}

func buildDNSEntryPoint(instance *v1alpha1.DynaKube) string {
	return "https://" + buildServiceName(instance.Name, module) + "." + instance.Namespace + ":9999/communication"
}
