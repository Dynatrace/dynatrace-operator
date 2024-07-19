package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/services"
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
		if err := r.removeService(ctx); err != nil {
			return errors.WithMessage(err, "could not remove service")
		}
	}

	return nil
}

func (r *reconciler) ensureService(ctx context.Context) error {
	_, err := getService(ctx, r.apiReader, r.dk.Name, r.dk.Namespace)
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

	newService, err := r.prepareService()
	if err != nil {
		return err
	}

	return r.client.Create(ctx, newService)
}

func (r *reconciler) removeService(ctx context.Context) error {
	log.Info("creating extension collector service")

	return services.NewQuery(ctx, r.client, r.apiReader, log).Delete(r.dk.Name, r.dk.Namespace)
}

func (r *reconciler) prepareService() (*corev1.Service, error) {
	coreLabels := labels.NewCoreLabels(r.dk.Name, labels.ExtensionComponentLabel)
	// TODO: add proper version later on
	appLabels := labels.NewAppLabels(labels.ExtensionComponentLabel, r.dk.Name, labels.ExtensionComponentLabel, "")

	newService, err := services.Create(r.dk,
		services.NewNameModifier(buildServiceName(r.dk.Name)),
		services.NewNamespaceModifier(r.dk.Namespace),
		services.NewPortsModifier(buildPortsName(r.dk.Name),
			ExtensionsCollectorComPort,
			corev1.ProtocolTCP,
			intstr.IntOrString{Type: 1, StrVal: ExtensionsCollectorTargetPortName},
		),
		services.NewLabelsModifier(coreLabels.BuildMatchLabels()),
	)

	newService.Spec.Selector = appLabels.BuildMatchLabels()

	return newService, err
}

func getService(ctx context.Context, apiReader client.Reader, dkName string, dkNamespace string) (*corev1.Service, error) {
	var svc corev1.Service

	err := apiReader.Get(ctx, client.ObjectKey{Name: buildServiceName(dkName), Namespace: dkNamespace}, &svc)
	if err != nil {
		return nil, err
	}

	return &svc, nil
}

func buildServiceName(dkName string) string {
	return dkName + ExtensionsControllerSuffix
}

func buildPortsName(dkName string) string {
	return dkName + ExtensionsControllerSuffix + "com-port"
}
