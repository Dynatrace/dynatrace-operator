package configsecret

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8sconditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	logMonitoringSecretSuffix = "-logmonitoring-config"

	TokenHashAnnotationKey   = api.InternalFlagPrefix + "tenant-token-hash"
	NetworkZoneAnnotationKey = api.InternalFlagPrefix + "network-zone"
	ProxyHashAnnotationKey   = api.InternalFlagPrefix + "proxy-hash"
	NoProxyAnnotationKey     = api.InternalFlagPrefix + "no-proxy"

	tenantKey       = "Tenant"
	tenantTokenKey  = "TenantToken"
	hostIDSourceKey = "HostIdSource"
	proxyKey        = "Proxy"
	noProxyKey      = "noProxy"
	serverKey       = "Server"
	networkZoneKey  = "Location"
)

type Reconciler struct {
	apiReader client.Reader
	dk        *dynakube.DynaKube
	secrets   k8ssecret.QueryObject
}

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dk *dynakube.DynaKube) *Reconciler {
	return &Reconciler{
		apiReader: apiReader,
		dk:        dk,
		secrets:   k8ssecret.Query(clt, apiReader, log),
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.LogMonitoring().IsStandalone() {
		if meta.FindStatusCondition(*r.dk.Conditions(), LmcConditionType) == nil {
			return nil // no condition == nothing is there to clean up
		}

		log.Info("cleaning up the LogMonitoring config-secret")

		err := r.secrets.Delete(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: GetSecretName(r.dk.Name), Namespace: r.dk.Namespace}})
		if err != nil {
			log.Error(err, "failed to clean-up LogMonitoring config-secret")
		}

		meta.RemoveStatusCondition(r.dk.Conditions(), LmcConditionType)

		return nil // clean-up shouldn't cause a failure
	}

	return r.reconcileSecret(ctx)
}

func (r *Reconciler) reconcileSecret(ctx context.Context) error {
	newSecret, err := r.prepareSecret(ctx)
	if err != nil {
		return err
	}

	changed, err := r.secrets.CreateOrUpdate(ctx, newSecret)
	if err != nil {
		k8sconditions.SetKubeAPIError(r.dk.Conditions(), LmcConditionType, err)

		return err
	} else if changed {
		k8sconditions.SetSecretOutdated(r.dk.Conditions(), LmcConditionType, newSecret.Name) // needed so the timestamp updates, will never actually show up in the status
	}

	k8sconditions.SetSecretCreated(r.dk.Conditions(), LmcConditionType, newSecret.Name)

	return nil
}

func (r *Reconciler) prepareSecret(ctx context.Context) (*corev1.Secret, error) {
	data, err := r.getSecretData(ctx)
	if err != nil {
		return nil, err
	}

	coreLabels := k8slabel.NewCoreLabels(r.dk.Name, k8slabel.LogMonitoringComponentLabel).BuildLabels()

	newSecret, err := k8ssecret.Build(r.dk,
		GetSecretName(r.dk.Name),
		data,
		k8ssecret.SetLabels(coreLabels),
	)
	if err != nil {
		log.Info("failed to build the final secret")

		k8sconditions.SetSecretGenFailed(r.dk.Conditions(), LmcConditionType, err)

		return nil, err
	}

	return newSecret, err
}

func (r *Reconciler) getSecretData(ctx context.Context) (map[string][]byte, error) {
	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{
		Name:      r.dk.OneAgent().GetTenantSecret(),
		Namespace: r.dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		log.Info("failed to get the oneagent-tenant secret")

		k8sconditions.SetKubeAPIError(r.dk.Conditions(), LmcConditionType, err)

		return nil, err
	}

	tenantUUID, err := r.dk.TenantUUID()
	if err != nil {
		log.Info("failed to determine the tenantUUID")

		k8sconditions.SetSecretGenFailed(r.dk.Conditions(), LmcConditionType, err)

		return nil, err
	}

	deploymentConfigContent := map[string]string{
		serverKey:       fmt.Sprintf("{%s}", r.dk.OneAgent().GetEndpoints()),
		tenantKey:       tenantUUID,
		tenantTokenKey:  tenantToken,
		hostIDSourceKey: "k8s-node-name",
	}

	if r.dk.HasProxy() {
		proxyURL, err := r.dk.Proxy(ctx, r.apiReader)
		if err != nil {
			log.Info("failed get the proxy value")

			k8sconditions.SetKubeAPIError(r.dk.Conditions(), LmcConditionType, err)

			return nil, err
		}

		deploymentConfigContent[proxyKey] = proxyURL
	}

	noProxy := createNoProxyValue(*r.dk)
	if noProxy != "" {
		deploymentConfigContent[noProxyKey] = noProxy
	}

	if r.dk.Spec.NetworkZone != "" {
		deploymentConfigContent[networkZoneKey] = r.dk.Spec.NetworkZone
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

func createNoProxyValue(dk dynakube.DynaKube) string {
	sources := []string{
		dk.FF().GetNoProxy(),
		capability.BuildHostEntries(dk),
	}

	noProxies := []string{}

	for _, source := range sources {
		if strings.TrimSpace(source) != "" {
			noProxies = append(noProxies, source)
		}
	}

	return strings.Join(noProxies, ",")
}

func GetSecretName(dkName string) string {
	return dkName + logMonitoringSecretSuffix
}

// AddAnnotations adds the key-values to the provided map for values within the secret that may change,
// and should cause the user of the secret to be restarted, if they don't read the config during runtime.
// Can't use a single hash for the config, as part of the secret (endpoints) changes too often.
func AddAnnotations(source map[string]string, dk dynakube.DynaKube) map[string]string {
	annotation := map[string]string{}
	if source != nil {
		annotation = maps.Clone(source)
	}

	annotation[TokenHashAnnotationKey] = dk.OneAgent().ConnectionInfo.TenantTokenHash

	if dk.Spec.NetworkZone != "" {
		annotation[NetworkZoneAnnotationKey] = dk.Spec.NetworkZone
	}

	if dk.HasProxy() {
		annotation[ProxyHashAnnotationKey] = dk.Status.ProxyURLHash
	}

	noProxy := createNoProxyValue(dk)
	if noProxy != "" {
		annotation[NoProxyAnnotationKey] = noProxy
	}

	return annotation
}
