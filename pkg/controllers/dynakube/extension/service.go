package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
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

		err = service.Query(r.client, r.apiReader, log).Delete(ctx, svc)
		if err != nil {
			log.Error(err, "failed to clean up extension service")

			return nil
		}

		return nil
	}

	return r.createOrUpdateService(ctx)
}

func (r *reconciler) createOrUpdateService(ctx context.Context) error {
	newService, err := r.buildService()
	if err != nil {
		conditions.SetServiceGenFailed(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	_, err = service.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update extension service")
		conditions.SetKubeAPIError(r.dk.Conditions(), serviceConditionType, err)

		return err
	}

	conditions.SetServiceCreated(r.dk.Conditions(), serviceConditionType, r.dk.Extensions().GetServiceName())

	return nil
}

func (r *reconciler) buildService() (*corev1.Service, error) {
	coreLabels := labels.NewCoreLabels(r.dk.Name, labels.ExtensionComponentLabel)
	appLabels := labels.NewAppLabels(labels.ExtensionComponentLabel, r.dk.Name, labels.ExtensionComponentLabel, "")

	svcPorts := []corev1.ServicePort{
		{
			Name:       r.dk.Extensions().GetPortName(),
			Port:       consts.ExtensionsCollectorTargetPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: consts.ExtensionsCollectorTargetPortName},
		},
	}

	return service.Build(r.dk,
		r.dk.Extensions().GetServiceName(),
		appLabels.BuildMatchLabels(),
		svcPorts,
		service.SetLabels(coreLabels.BuildLabels()),
		service.SetType(corev1.ServiceTypeClusterIP),
	)
}
