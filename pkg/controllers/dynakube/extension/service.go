package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/service"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *reconciler) reconcileService(ctx context.Context) error {
	if r.dk.PrometheusEnabled() {
		log.Info("reconcile service")

		if err := r.ensureService(ctx); err != nil {
			return errors.WithMessage(err, "could not update service")
		}

		return nil
	} else {
		svc, err := r.buildService()
		if err != nil {
			log.Error(err, "could not build service during cleanup")
		}

		err = service.Query(r.client, r.apiReader, log).Delete(ctx, svc)

		if err != nil {
			log.Error(err, "failed to clean up extension service")

			return nil
		}
	}

	return nil
}

func (r *reconciler) ensureService(ctx context.Context) error {
	_, err := service.Query(r.client, r.apiReader, log).Get(ctx, client.ObjectKey{Name: r.buildServiceName(), Namespace: r.dk.Namespace})

	if k8serrors.IsNotFound(err) {
		log.Info("service was not found, creating service")

		return r.createService(ctx)
	} else if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) createService(ctx context.Context) error {
	log.Info("creating extension collector service")

	newService, err := r.buildService()
	if err != nil {
		return err
	}

	_, err = service.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)

	return err
}

func (r *reconciler) buildService() (*corev1.Service, error) {
	coreLabels := labels.NewCoreLabels(r.dk.Name, labels.ExtensionComponentLabel)
	// TODO: add proper version later on
	appLabels := labels.NewAppLabels(labels.ExtensionComponentLabel, r.dk.Name, labels.ExtensionComponentLabel, "")

	svcPort := corev1.ServicePort{
		Name:       buildPortsName(r.dk.Name),
		Port:       ExtensionsCollectorComPort,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.IntOrString{Type: 1, StrVal: ExtensionsCollectorTargetPortName},
	}

	return service.Build(r.dk,
		r.buildServiceName(),
		appLabels.BuildMatchLabels(),
		svcPort,
		service.SetLabels(coreLabels.BuildMatchLabels()),
		service.SetType(corev1.ServiceTypeClusterIP),
	)
}

func (r *reconciler) buildServiceName() string {
	return r.dk.Name + ExtensionsControllerSuffix
}

func buildPortsName(dkName string) string {
	return dkName + ExtensionsControllerSuffix + "com-port"
}
