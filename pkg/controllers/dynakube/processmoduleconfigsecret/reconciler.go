package processmoduleconfigsecret

import (
	"context"
	"encoding/json"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/timeprovider"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
		log.Info("skipping process module secret reconciler")
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

func (r *Reconciler) getOrCreateSecretIfNotExists(ctx context.Context) (*corev1.Secret, error) { // created, error
	var config corev1.Secret
	err := r.apiReader.Get(ctx, client.ObjectKey{Name: extendWithSuffix(r.dynakube.Name), Namespace: r.dynakube.Namespace}, &config)
	if k8serrors.IsNotFound(err) {
		log.Info("creating pull secret")
		newSecret, err := r.prepareSecret()
		if err != nil {
			return nil, err
		}

		if err = r.client.Create(ctx, newSecret); err != nil {
			return nil, err
		}

		r.dynakube.Status.OneAgent.LastProcessModuleConfigUpdate = r.timeProvider.Now()
		return newSecret, nil
	}
	return &config, nil
}

func (r *Reconciler) updateSecret(ctx context.Context, oldSecret *corev1.Secret) error {
	newSecret, err := r.prepareSecret()
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

func (r *Reconciler) prepareSecret() (*corev1.Secret, error) {
	pmc, err := r.dtClient.GetProcessModuleConfig(0)
	if err != nil {
		return nil, err
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

func extendWithSuffix(name string) string {
	return name + PullSecretSuffix
}
