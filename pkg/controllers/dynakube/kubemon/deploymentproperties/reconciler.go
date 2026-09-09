// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package deploymentproperties

import (
	"context"
	"fmt"
	"strings"

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

	if dk.KubernetesMonitoring().IsEnabled() && dk.KubernetesMonitoring().NeedsDeploymentProperties() {
		return r.createDeploymentPropertiesSecret(ctx, dk)
	} else {
		return client.IgnoreNotFound(r.client.Delete(ctx, secretSpec(dk, nil)))
	}
}

func (r *Reconciler) createDeploymentPropertiesSecret(ctx context.Context, dk *dynakube.DynaKube) error {
	desired := secretSpec(dk, secretData(dk))

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	return k8sobject.RetryCreateOrUpdate(ctx, r.client, secret, func() error {
		secret.Labels = desired.Labels
		secret.Data = desired.Data

		return controllerutil.SetControllerReference(dk, secret, r.client.Scheme())
	})
}

func secretData(dk *dynakube.DynaKube) map[string][]byte {
	data := ""

	if len(dk.GetResourceAttributes()) > 0 {
		data += deploymentproperties.BuildContent(dk.GetResourceAttributes())
	}
data += "\n"
	if dk.NeedsCustomNoProxy() {
		noProxyValue := strings.ReplaceAll(dk.FF().GetNoProxy(), ",", "|")
		data += fmt.Sprintf("%s\n%s=%s", agconsts.PropertiesClientInternalSection, agconsts.PropertiesNoProxyFieldName, noProxyValue)
	}

	return map[string][]byte{
		agconsts.DeploymentPropertiesFileName: []byte(data),
	}
}

func secretSpec(dk *dynakube.DynaKube, data map[string][]byte) *corev1.Secret {
	labels := k8slabel.New(k8slabel.KubeMonComponentLabel, dk.Name, "")

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dk.KubernetesMonitoring().GetDeploymentPropertiesSecretName(),
			Namespace: dk.Namespace,
			Labels:    labels.AsMap(),
		},
		Data: data,
	}
}
