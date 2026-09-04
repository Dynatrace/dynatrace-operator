package deploymentproperties

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	agconsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/deploymentproperties"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciler struct {
	client client.Client
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		client: kubeClient,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, dk *dynakube.DynaKube) error {
	ctx, _ = logd.NewFromContext(ctx, "deploymentproperties")

	if dk.KubernetesMonitoring().IsEnabled() && len(dk.GetResourceAttributes()) > 0 {
		return r.createDeploymentPropertiesConfigMap(ctx, dk)
	} else {
		return client.IgnoreNotFound(r.client.Delete(ctx, configMapSpec(dk, nil)))
	}
}

func (r *Reconciler) createDeploymentPropertiesConfigMap(ctx context.Context, dk *dynakube.DynaKube) error {
	desired := configMapSpec(dk, ConfigMapData(dk))

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	return k8sobject.RetryCreateOrUpdate(ctx, r.client, configMap, func() error {
		configMap.Labels = desired.Labels
		configMap.Data = desired.Data

		return controllerutil.SetControllerReference(dk, configMap, r.client.Scheme())
	})
}

func ConfigMapData(dk *dynakube.DynaKube) map[string]string {
	return map[string]string{
		agconsts.DeploymentPropertiesFileName: deploymentproperties.BuildContent(dk.Spec.ResourceAttributes),
	}
}

func configMapSpec(dk *dynakube.DynaKube, data map[string]string) *corev1.ConfigMap {
	labels := k8slabel.New(k8slabel.KubeMonComponentLabel, dk.Name, "")

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetDeploymentPropertiesConfigMapName(),
			Namespace: dk.Namespace,
			Labels:    labels.AsMap(),
		},
		Data: data,
	}
}
