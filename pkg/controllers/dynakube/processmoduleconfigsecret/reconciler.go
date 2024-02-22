package processmoduleconfigsecret

import (
	"context"
	"encoding/json"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	dynakube     *dynatracev1beta1.DynaKube
	scheme       *runtime.Scheme
	timeProvider *timeprovider.Provider
}

func NewReconciler(clt client.Client, //nolint:revive
	apiReader client.Reader,
	dtClient dtclient.Client,
	dynakube *dynatracev1beta1.DynaKube,
	scheme *runtime.Scheme,
	timeProvider *timeprovider.Provider) *Reconciler {
	return &Reconciler{
		client:       clt,
		apiReader:    apiReader,
		dtClient:     dtClient,
		dynakube:     dynakube,
		scheme:       scheme,
		timeProvider: timeProvider,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dynakube.CloudNativeFullstackMode() || r.dynakube.ApplicationMonitoringMode() {
		err := r.reconcileSecret(ctx)
		if err != nil {
			log.Info("could not reconcile pull secret")

			return errors.WithStack(err)
		}
	} else {
		log.Info("skipping process module config secret reconciler")
	}

	return nil
}

func (r *Reconciler) reconcileSecret(ctx context.Context) error {
	secret, err := r.getOrCreateSecretIfNotExists(ctx)
	if err != nil {
		return errors.WithMessage(err, "could not get or create secret")
	}

	if err := r.updateSecretIfOutdated(ctx, secret); err != nil {
		return errors.WithMessage(err, "could not update secret")
	}

	return nil
}

func (r *Reconciler) getOrCreateSecretIfNotExists(ctx context.Context) (*corev1.Secret, error) {
	config, err := getSecret(ctx, r.apiReader, r.dynakube.Name, r.dynakube.Namespace)
	if k8serrors.IsNotFound(err) {
		log.Info("creating process module config secret")

		newSecret, err := r.prepareSecret(ctx)
		if err != nil {
			return nil, err
		}

		if err = r.client.Create(ctx, newSecret); err != nil {
			return nil, err
		}

		r.dynakube.Status.OneAgent.LastProcessModuleConfigUpdate = r.timeProvider.Now()

		return newSecret, nil
	} else if err != nil {
		return nil, err
	}

	return config, nil
}

func (r *Reconciler) updateSecret(ctx context.Context, oldSecret *corev1.Secret) error {
	newSecret, err := r.prepareSecret(ctx)
	if err != nil {
		return err
	}

	oldSecret.Data = newSecret.Data
	if err = r.client.Update(ctx, oldSecret); err != nil {
		return err
	}

	r.dynakube.Status.OneAgent.LastProcessModuleConfigUpdate = r.timeProvider.Now()

	return nil
}

func (r *Reconciler) updateSecretIfOutdated(ctx context.Context, oldSecret *corev1.Secret) error {
	if r.timeProvider.IsOutdated(r.dynakube.Status.OneAgent.LastProcessModuleConfigUpdate, r.dynakube.FeatureApiRequestThreshold()) {
		return r.updateSecret(ctx, oldSecret)
	} else {
		log.Info("skipping updating process module config due to min request threshold")
	}

	return nil
}

func (r *Reconciler) prepareSecret(ctx context.Context) (*corev1.Secret, error) {
	pmc, err := r.dtClient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		return nil, err
	}

	tenantToken, err := secret.GetDataFromSecretName(r.apiReader, types.NamespacedName{
		Name:      r.dynakube.OneagentTenantSecret(),
		Namespace: r.dynakube.Namespace,
	}, connectioninfo.TenantTokenName, log)
	if err != nil {
		return nil, err
	}

	pmc = pmc.
		AddHostGroup(r.dynakube.HostGroup()).
		AddConnectionInfo(r.dynakube.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if r.dynakube.NeedsOneAgentProxy() {
		proxy, err := r.dynakube.Proxy(ctx, r.apiReader)
		if err != nil {
			return nil, err
		}

		pmc.AddProxy(proxy)

		if r.dynakube.NeedsActiveGate() {
			multiCap := capability.NewMultiCapability(r.dynakube)
			pmc.AddNoProxy(capability.BuildDNSEntryPointWithoutEnvVars(r.dynakube.Name, r.dynakube.Namespace, multiCap))
		}
	}

	marshaled, err := json.Marshal(pmc)
	if err != nil {
		log.Info("could not marshal process module config")

		return nil, err
	}

	newSecret, err := secret.Create(r.scheme, r.dynakube,
		secret.NewNameModifier(extendWithSuffix(r.dynakube.Name)),
		secret.NewNamespaceModifier(r.dynakube.Namespace),
		secret.NewTypeModifier(corev1.SecretTypeOpaque),
		secret.NewDataModifier(map[string][]byte{SecretKeyProcessModuleConfig: marshaled}))

	return newSecret, err
}

func GetSecretData(ctx context.Context, apiReader client.Reader, dynakubeName string, dynakubeNamespace string) (*dtclient.ProcessModuleConfig, error) {
	secret, err := getSecret(ctx, apiReader, dynakubeName, dynakubeNamespace)
	if err != nil {
		return nil, err
	}

	processModuleConfig, err := unmarshal(secret)
	if err != nil {
		return nil, err
	}

	return processModuleConfig, nil
}

func getSecret(ctx context.Context, apiReader client.Reader, dynakubeName string, dynakubeNamespace string) (*corev1.Secret, error) {
	var config corev1.Secret

	err := apiReader.Get(ctx, client.ObjectKey{Name: extendWithSuffix(dynakubeName), Namespace: dynakubeNamespace}, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
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
