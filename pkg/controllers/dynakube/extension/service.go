package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sservice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *reconciler) reconcileService(ctx context.Context) error {
	if !r.dk.Extensions().IsAnyEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), serviceConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), serviceConditionType)

		svc, err := r.buildService()
		if err != nil {
			log.Error(err, "could not build service during cleanup")

			return err
		}

		err = k8sservice.Query(r.client, r.apiReader, log).Delete(ctx, svc)
		if err != nil {
			log.Error(err, "failed to clean up extension service")
		}

		r.deleteLegacyService(ctx)

		return nil
	}

	defer r.deleteLegacyService(ctx)

	return r.createOrUpdateService(ctx)
}

func (r *reconciler) createOrUpdateService(ctx context.Context) error {
	newService, err := r.buildService()
	if err != nil {
		conditions.SetServiceGenFailed(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	_, err = k8sservice.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update extension service")
		conditions.SetKubeAPIError(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	conditions.SetServiceCreated(r.dk.Conditions(), serviceConditionType, r.dk.Extensions().GetServiceName())

	return nil
}

func (r *reconciler) buildService() (*corev1.Service, error) {
	coreLabels := k8slabel.NewCoreLabels(r.dk.Name, k8slabel.ExtensionComponentLabel)
	appLabels := k8slabel.NewAppLabels(k8slabel.ExtensionComponentLabel, r.dk.Name, k8slabel.ExtensionComponentLabel, "")

	svcPorts := []corev1.ServicePort{
		{
			Name:       r.dk.Extensions().GetPortName(),
			Port:       consts.ExtensionsDatasourceTargetPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: consts.ExtensionsDatasourceTargetPortName},
		},
	}

	return k8sservice.Build(r.dk,
		r.dk.Extensions().GetServiceName(),
		appLabels.BuildMatchLabels(),
		svcPorts,
		k8sservice.SetLabels(coreLabels.BuildLabels()),
		k8sservice.SetType(corev1.ServiceTypeClusterIP),
	)
}

// TODO: Remove as part of DAQ-18375
func (r *reconciler) deleteLegacyService(ctx context.Context) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.dk.Name + "-extensions-controller",
			Namespace: r.dk.Namespace,
		},
	}

	_ = r.client.Delete(ctx, svc)
}
