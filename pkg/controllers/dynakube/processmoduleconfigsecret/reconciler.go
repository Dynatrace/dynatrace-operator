package processmoduleconfigsecret

import (
	"context"
	"encoding/json"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube"
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
	SecretSuffix                 = "-pmc-secret"
	SecretKeyProcessModuleConfig = "ruxitagentproc.conf"
)

type Reconciler struct {
	client       client.Client
	apiReader    client.Reader
	dtClient     dtclient.Client
	dk           *dynakube.DynaKube
	timeProvider *timeprovider.Provider
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

	return r
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	isNeeded := r.dk.OneAgent().IsCSIAvailable() &&
		(r.dk.OneAgent().IsCloudNativeFullstackMode() ||
			r.dk.OneAgent().IsApplicationMonitoringMode())

	if !(isNeeded) {
		if meta.FindStatusCondition(*r.dk.Conditions(), PMCConditionType) == nil {
			return nil
		}

		defer meta.RemoveStatusCondition(r.dk.Conditions(), PMCConditionType)

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
		return errors.WithMessage(err, "failed to create or update processModuleConfig secret")
	}

	return nil
}

func (r *Reconciler) reconcileSecret(ctx context.Context) error {
	if !conditions.IsOutdated(r.timeProvider, r.dk, PMCConditionType) {
		return nil
	}

	log.Info("processModuleConfig is outdated, updating")
	conditions.SetSecretOutdated(r.dk.Conditions(), PMCConditionType, "secret is outdated, update in progress")

	secret, err := r.prepareSecret(ctx)
	if err != nil {
		return err
	}

	return r.createOrUpdateSecret(ctx, secret)
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, secret *corev1.Secret) error {
	_, err := k8ssecret.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, secret)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), PMCConditionType, err)

		return errors.Errorf("failed to create or update secret '%s': %v", secret.Name, err)
	}

	conditions.SetSecretCreatedOrUpdated(r.dk.Conditions(), PMCConditionType, secret.Name)

	return nil
}

func (r *Reconciler) deleteSecret(ctx context.Context, secret *corev1.Secret) error {
	if err := k8ssecret.Query(r.client, r.apiReader, log).Delete(ctx, secret); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), PMCConditionType, err)

		return err
	}

	return nil
}

func (r *Reconciler) prepareSecret(ctx context.Context) (*corev1.Secret, error) {
	pmc, err := r.dtClient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		conditions.SetDynatraceApiError(r.dk.Conditions(), PMCConditionType, err)

		return nil, err
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, r.apiReader, types.NamespacedName{
		Name:      r.dk.OneAgent().GetTenantSecret(),
		Namespace: r.dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), PMCConditionType, err)

		return nil, err
	}

	pmc = pmc.
		AddHostGroup(r.dk.OneAgent().GetHostGroup()).
		AddConnectionInfo(r.dk.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if r.dk.NeedsOneAgentProxy() {
		proxy, err := r.dk.Proxy(ctx, r.apiReader)
		if err != nil {
			conditions.SetKubeApiError(r.dk.Conditions(), PMCConditionType, err)

			return nil, err
		}

		pmc.AddProxy(proxy)

		multiCap := capability.NewMultiCapability(r.dk)
		dnsEntry := capability.BuildDNSEntryPointWithoutEnvVars(r.dk.Name, r.dk.Namespace, multiCap)

		if r.dk.FF().GetNoProxy() != "" {
			dnsEntry += "," + r.dk.FF().GetNoProxy()
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
		conditions.SetKubeApiError(r.dk.Conditions(), PMCConditionType, err)

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
	return name + SecretSuffix
}
