package configsecret

import (
	"context"
	"fmt"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	logMonitoringSecretSuffix = "-logmonitoring-config"

	tenantKey       = "Tenant"
	tenantTokenKey  = "TenantToken"
	hostIdSourceKey = "HostIdSource"
	serverKey       = "Server"
)

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.LogMonitoring().IsEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), lmcConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		query := k8ssecret.Query(r.client, r.apiReader, log)
		err := query.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: GetSecretName(r.dk.Name), Namespace: r.dk.Namespace}})

		if err != nil {
			log.Error(err, "failed to clean-up LogMonitoring config-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), lmcConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	return r.reconcileSecret(ctx)
}

func (r *Reconciler) reconcileSecret(ctx context.Context) error {
	query := k8ssecret.Query(r.client, r.apiReader, log)

	newSecret, err := r.prepareSecret(ctx)
	if err != nil {
		return err
	}

	changed, err := query.CreateOrUpdate(ctx, newSecret)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), lmcConditionType, err)

		return err
	} else if changed {
		conditions.SetSecretOutdated(r.dk.Conditions(), lmcConditionType, newSecret.Name) // needed so the timestamp updates, will never actually show up in the status
	}

	conditions.SetSecretCreated(r.dk.Conditions(), lmcConditionType, newSecret.Name)

	return nil
}

func (r *Reconciler) prepareSecret(ctx context.Context) (*corev1.Secret, error) {
	data, err := r.getSecretData(ctx)
	if err != nil {
		return nil, err
	}

	coreLabels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.LogMonitoringComponentLabel).BuildLabels()

	newSecret, err := k8ssecret.Build(r.dk,
		GetSecretName(r.dk.Name),
		data,
		k8ssecret.SetLabels(coreLabels),
	)
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), lmcConditionType, err)

		return nil, err
	}

	return newSecret, err
}

func (r *Reconciler) getSecretData(ctx context.Context) (map[string][]byte, error) {
	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{
		Name:      r.dk.OneagentTenantSecret(),
		Namespace: r.dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), lmcConditionType, err)

		return nil, err
	}

	tenantUUID, err := r.dk.TenantUUIDFromConnectionInfoStatus()
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), lmcConditionType, err)

		return nil, err
	}

	deploymentConfigContent := map[string]string{
		serverKey:       fmt.Sprintf("{%s}", r.dk.OneAgentEndpoints()),
		tenantKey:       tenantUUID,
		tenantTokenKey:  tenantToken,
		hostIdSourceKey: "k8s-node-name",
	}

	var content strings.Builder
	for key, value := range deploymentConfigContent {
		content.WriteString(key)
		content.WriteString("=")
		content.WriteString(value)
		content.WriteString("\n")
	}

	return map[string][]byte{DeploymentConfigFilename: []byte(content.String())}, nil
}

func GetSecretName(dkName string) string {
	return dkName + logMonitoringSecretSuffix
}
