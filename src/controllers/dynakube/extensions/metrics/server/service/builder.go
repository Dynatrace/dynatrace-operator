package service

import (
	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/extensions/metrics/common"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects/address"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	regv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

const (
	httpsServicePort = 443
	httpServicePort  = 80

	apiServiceGroupPriorityMinimum = 100
	apiServiceVersionPriority      = 100
)

type builder struct {
	*dynatracev1beta1.DynaKube
	*appsv1.Deployment
}

func newBuilder(
	dynaKube *dynatracev1beta1.DynaKube,
	deployment *appsv1.Deployment,
) *builder {
	return &builder{
		DynaKube:   dynaKube,
		Deployment: deployment,
	}
}

func (builder *builder) newService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      builder.Deployment.Name,
			Namespace: builder.DynaKube.Namespace,
			Labels:    builder.Deployment.ObjectMeta.Labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: builder.Deployment.Spec.Selector.MatchLabels,

			Ports: []corev1.ServicePort{
				{
					Name:       common.HttpsServicePortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       httpsServicePort,
					TargetPort: intstr.FromString(common.HttpsServicePortName),
				},
				{
					Name:       common.HttpServicePortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       httpServicePort,
					TargetPort: intstr.FromString(common.HttpServicePortName),
				},
			},
		},
	}
}

func (builder *builder) newApiService() *regv1.APIService {
	return &regv1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.ApiServiceVersionGroup,
			Annotations: map[string]string{
				common.ControlledByDynaKubeAnnotation: builder.DynaKube.Name,
			},
		},
		Spec: regv1.APIServiceSpec{
			Service: &regv1.ServiceReference{
				Name:      builder.Deployment.Name,
				Namespace: builder.DynaKube.Namespace,
				Port:      address.Of[int32](httpsServicePort),
			},

			Group:                 common.ApiServiceGroup,
			Version:               common.ApiServiceVersion,
			InsecureSkipTLSVerify: true,
			GroupPriorityMinimum:  apiServiceGroupPriorityMinimum,
			VersionPriority:       apiServiceVersionPriority,
		},
	}
}
