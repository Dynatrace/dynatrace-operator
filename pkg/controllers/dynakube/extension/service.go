package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/service"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *reconciler) reconcileService(ctx context.Context) error {
	if !r.dk.PrometheusEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), extensionsServiceConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), extensionsServiceConditionType)

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
	log.Info("creating extension collector service")

	newService, err := r.buildService()
	if err != nil {
		conditions.SetServiceGenFailed(r.dk.Conditions(), extensionsServiceConditionType, err)

		return err
	}

	_, err = service.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update extension service")
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsServiceConditionType, err)

		return err
	}

	conditions.SetServiceCreated(r.dk.Conditions(), extensionsServiceConditionType, r.buildServiceName())

	return nil
}

func (r *reconciler) buildService() (*corev1.Service, error) {
	coreLabels := labels.NewCoreLabels(r.dk.Name, labels.ExtensionComponentLabel)
	// TODO: add proper version later on
	appLabels := labels.NewAppLabels(labels.ExtensionComponentLabel, r.dk.Name, labels.ExtensionComponentLabel, "")

	svcPort := corev1.ServicePort{
		Name:       r.buildPortsName(),
		Port:       ExtensionsCollectorComPort,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: ExtensionsCollectorTargetPortName},
	}

	return service.Build(r.dk,
		r.buildServiceName(),
		appLabels.BuildMatchLabels(),
		svcPort,
		service.SetLabels(coreLabels.BuildLabels()),
		service.SetType(corev1.ServiceTypeClusterIP),
	)
}

func (r *reconciler) buildServiceName() string {
	return r.dk.Name + ExtensionsControllerSuffix
}

func (r *reconciler) buildPortsName() string {
	return r.dk.Name + ExtensionsControllerSuffix + "com-port"
}
