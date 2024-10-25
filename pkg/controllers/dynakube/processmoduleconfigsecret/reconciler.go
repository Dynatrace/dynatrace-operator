package processmoduleconfigsecret

import (
	"context"
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	PullSecretSuffix             = "-pmc-secret"
	SecretKeyProcessModuleConfig = "ruxitagentproc.conf"
)

type Reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dtClient     dtclient.Client
	dk           *dynakube.DynaKube
	timeProvider *timeprovider.Provider
	secretQuery  k8ssecret.QueryObject
}

func NewReconciler(clt client.Client,
	apiReader client.Reader,
	dtClient dtclient.Client,
	dk *dynakube.DynaKube,
	timeProvider *timeprovider.Provider) *Reconciler {
	r := &Reconciler{
		client:       clt,
		apiReader:    apiReader,
		dtClient:     dtClient,
		dk:           dk,
		timeProvider: timeProvider,
	}
	r.secretQuery = k8ssecret.Query(clt, apiReader, log)

	return r
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !(r.dk.CloudNativeFullstackMode() || r.dk.ApplicationMonitoringMode()) {
		if meta.FindStatusCondition(*r.dk.Conditions(), pmcConditionType) == nil {
			return nil
		}

		defer meta.RemoveStatusCondition(r.dk.Conditions(), pmcConditionType)

		err := r.deleteSecret(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      extendWithSuffix(r.dk.Name),
				Namespace: r.dk.Namespace,
			},
		})

		if err != nil {
			return errors.WithMessage(err, "failed to delete processModuleConfig secret")
		}

		return nil
	}

	err := r.reconcileSecret(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to create processModuleConfig secret")
	}

	return nil
}

func (r *Reconciler) reconcileSecret(ctx context.Context) error {
	if !conditions.IsOutdated(r.timeProvider, r.dk, pmcConditionType) {
		return nil
	}

	log.Info("processModuleConfig is outdated, updating")
	conditions.SetSecretOutdated(r.dk.Conditions(), pmcConditionType, "secret is outdated, update in progress")

	secret, err := r.prepareSecret(ctx)
	if err != nil {
		return err
	}

	return r.createOrUpdateSecret(ctx, secret)
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret) error {
	_, err := r.secretQuery.WithOwner(r.dk).CreateOrUpdate(ctx, secret)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), pmcConditionType, err)

		return errors.Errorf("failed to create secret '%s': %v", secret.Name, err)
	}

	conditions.SetSecretCreatedOrUpdated(r.dk.Conditions(), pmcConditionType, secret.Name)

	return nil
}

func (r *Reconciler) deleteSecret(ctx context.Context, secret *corev1.Secret) error {
	if err := r.secretQuery.Delete(ctx, secret); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), pmcConditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) prepareSecret(ctx context.Context) (*corev1.Secret, error) {
	pmc, err := r.dtClient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		conditions.SetDynatraceApiError(r.dk.Conditions(), pmcConditionType, err)

		return nil, err
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{
		Name:      r.dk.OneagentTenantSecret(),
		Namespace: r.dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), pmcConditionType, err)

		return nil, err
	}

	pmc = pmc.
		AddHostGroup(r.dk.HostGroup()).
		AddConnectionInfo(r.dk.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if r.dk.NeedsOneAgentProxy() {
		proxy, err := r.dk.Proxy(ctx, r.apiReader)
		if err != nil {
			conditions.SetKubeApiError(r.dk.Conditions(), pmcConditionType, err)

			return nil, err
		}

		pmc.AddProxy(proxy)

		multiCap := capability.NewMultiCapability(r.dk)
		dnsEntry := capability.BuildDNSEntryPointWithoutEnvVars(r.dk.Name, r.dk.Namespace, multiCap)

		if r.dk.FeatureNoProxy() != "" {
			dnsEntry += "," + r.dk.FeatureNoProxy()
		}

		pmc.AddNoProxy(dnsEntry)
	}

	marshaled, err := json.Marshal(pmc)
	if err != nil {
		log.Info("could not marshal process module config")

		return nil, err
	}

	newSecret, err := k8ssecret.Build(r.dk,
		extendWithSuffix(r.dk.Name),
		map[string][]byte{SecretKeyProcessModuleConfig: marshaled})

	k8ssecret.SetType(corev1.SecretTypeOpaque)

	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), pmcConditionType, err)

		return nil, err
	}

	return newSecret, err
}

func GetSecretData(ctx context.Context, apiReader client.Reader, dynakubeName string, dynakubeNamespace string) (*dtclient.ProcessModuleConfig, error) {
	typedName := types.NamespacedName{Namespace: dynakubeNamespace, Name: extendWithSuffix(dynakubeName)}

	secret, err := k8ssecret.Query(nil, apiReader, log).Get(ctx, typedName)
	if err != nil {
		return nil, err
	}

	processModuleConfig, err := unmarshal(secret)
	if err != nil {
		return nil, err
	}

	return processModuleConfig, nil
}

func unmarshal(secret *corev1.Secret) (*dtclient.ProcessModuleConfig, error) {
	var config *dtclient.ProcessModuleConfig

	err := json.Unmarshal(secret.Data[SecretKeyProcessModuleConfig], &config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func extendWithSuffix(name string) string {
	return name + PullSecretSuffix
}
