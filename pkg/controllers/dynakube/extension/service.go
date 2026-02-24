package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sservice"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (r *Reconciler) reconcileService(ctx context.Context, dk *dynakube.DynaKube) error {
	if !dk.Extensions().IsAnyEnabled() {
		if meta.FindStatusCondition(*dk.Conditions(), serviceConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(dk.Conditions(), serviceConditionType)

		svc, err := r.buildService(dk)
		if err != nil {
			log.Error(err, "could not build service during cleanup")

			return err
		}

		err = k8sservice.Query(r.client, r.apiReader, log).Delete(ctx, svc)
		if err != nil {
			log.Error(err, "failed to clean up extension service")
		}

		r.deleteLegacyService(ctx, dk)

		return nil
	}

	defer r.deleteLegacyService(ctx, dk)

	return r.createOrUpdateService(ctx, dk)
}

func (r *Reconciler) createOrUpdateService(ctx context.Context, dk *dynakube.DynaKube) error {
	newService, err := r.buildService(dk)
	if err != nil {
		k8sconditions.SetServiceGenFailed(dk.Conditions(), serviceConditionType, err)

		return err
	}

	_, err = k8sservice.Query(r.client, r.apiReader, log).CreateOrUpdate(ctx, newService)
	if err != nil {
		log.Info("failed to create/update extension service")
		k8sconditions.SetKubeAPIError(dk.Conditions(), serviceConditionType, err)

		return err
	}

	k8sconditions.SetServiceCreated(dk.Conditions(), serviceConditionType, dk.Extensions().GetServiceName())

	return nil
}

func (r *Reconciler) buildService(dk *dynakube.DynaKube) (*corev1.Service, error) {
	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ExtensionComponentLabel)
	appLabels := k8slabel.NewAppLabels(k8slabel.ExtensionComponentLabel, dk.Name, k8slabel.ExtensionComponentLabel, "")

	svcPorts := []corev1.ServicePort{
		{
			Name:       dk.Extensions().GetPortName(),
			Port:       consts.ExtensionsDatasourceTargetPort,
			Protocol:   corev1.ProtocolTCP,
			TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: consts.ExtensionsDatasourceTargetPortName},
		},
	}

	return k8sservice.Build(dk,
		dk.Extensions().GetServiceName(),
		appLabels.BuildMatchLabels(),
		svcPorts,
		k8sservice.SetLabels(coreLabels.BuildLabels()),
		k8sservice.SetType(corev1.ServiceTypeClusterIP),
	)
}

// TODO: Remove as part of DAQ-18375
func (r *Reconciler) deleteLegacyService(ctx context.Context, dk *dynakube.DynaKube) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.Name + "-extensions-controller",
			Namespace: dk.Namespace,
		},
	}

	_ = r.client.Delete(ctx, svc)
}
