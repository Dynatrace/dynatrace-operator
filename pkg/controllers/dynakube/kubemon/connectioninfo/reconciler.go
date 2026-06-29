package connectioninfo

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/shared/communication"
	agclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace/activegate"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sconfigmap"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	secrets    k8ssecret.QueryObject
	configMaps k8sconfigmap.QueryObject
}

func NewReconciler(kubeClient client.Client) *Reconciler {
	return &Reconciler{
		secrets:    k8ssecret.Query(kubeClient, kubeClient),
		configMaps: k8sconfigmap.Query(kubeClient, kubeClient),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, agClient agclient.Client, dk *dynakube.DynaKube) error {
	if !dk.KubernetesMonitoring().IsEnabled() {
		return r.cleanup(ctx, dk)
	}

	info, err := agClient.GetConnectionInfo(ctx)
	if err != nil {
		return err
	}

	if info.Endpoints == "" {
		return errors.New("kubemon connection info has no endpoints yet")
	}

	if err := r.createOrUpdateConfigMap(ctx, dk, info); err != nil {
		return err
	}

	if err := r.createOrUpdateSecret(ctx, dk, info.TenantToken); err != nil {
		return err
	}

	dk.Status.KubernetesMonitoring.ConnectionInfo.TenantUUID = info.TenantUUID
	dk.Status.KubernetesMonitoring.ConnectionInfo.Endpoints = info.Endpoints

	return nil
}

func (r *Reconciler) cleanup(ctx context.Context, dk *dynakube.DynaKube) error {
	cmName := dk.KubernetesMonitoring().GetConnectionInfoConfigMapName()
	if err := r.configMaps.Delete(ctx, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: dk.Namespace}}); err != nil {
		return err
	}

	secretName := dk.KubernetesMonitoring().GetTenantSecretName()
	if err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: dk.Namespace}}); err != nil {
		return err
	}

	dk.Status.KubernetesMonitoring.ConnectionInfo = communication.ConnectionInfo{}

	return nil
}

func (r *Reconciler) createOrUpdateConfigMap(ctx context.Context, dk *dynakube.DynaKube, info agclient.ConnectionInfo) error {
	data := map[string]string{}
	if info.TenantUUID != "" {
		data[connectioninfo.TenantUUIDKey] = info.TenantUUID
	}

	if info.Endpoints != "" {
		data[connectioninfo.CommunicationEndpointsKey] = info.Endpoints
	}

	coreLabels := k8slabel.NewCoreLabels(dk.Name, k8slabel.ActiveGateComponentLabel)

	cm, err := k8sconfigmap.Build(dk,
		dk.KubernetesMonitoring().GetConnectionInfoConfigMapName(),
		data,
		k8sconfigmap.SetLabels(coreLabels.BuildLabels()),
	)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.configMaps.CreateOrUpdate(ctx, cm)
	if err != nil {
		return err
	}

	return nil
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, dk *dynakube.DynaKube, tenantToken string) error {
	secret, err := connectioninfo.BuildTenantSecret(dk, k8slabel.ActiveGateComponentLabel, dk.KubernetesMonitoring().GetTenantSecretName(), tenantToken)
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.secrets.CreateOrUpdate(ctx, secret)

	return err
}
