package ncctoken

import (
	"context"
	"strconv"
	"strings"
	"time"

	dynatracev1beta2 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta2/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NodeConfigCollectorTokenName = "kspm-node-collector-token"

	// Buffer to avoid warnings in the UI
	AuthTokenBuffer           = time.Hour * 24
	AuthTokenRotationInterval = time.Hour*24*30 - AuthTokenBuffer
)

var _ controllers.Reconciler = &Reconciler{}

type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dynakube  *dynatracev1beta2.DynaKube
	dtc       dtclient.Client
}

func NewReconciler(clt client.Client, apiReader client.Reader, dynakube *dynatracev1beta2.DynaKube, dtc dtclient.Client) *Reconciler {
	return &Reconciler{
		client:    clt,
		apiReader: apiReader,
		dynakube:  dynakube,
		dtc:       dtc,
	}
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if !r.dynakube.NeedsActiveGate() {
		_ = meta.RemoveStatusCondition(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType)

		return nil
	}

	err := r.reconcileNodeConfigCollectorTokenSecret(ctx)
	if err != nil {
		return errors.WithMessage(err, "failed to create Node Config Collector secret")
	}

	return nil
}

func (r *Reconciler) ensureNodeConfigCollectorTokenSecret(ctx context.Context) error {
	tokenSecretData, err := r.generateNodeConfigCollectorToken(ctx)
	if err != nil {
		return errors.WithMessagef(err, "failed to create secret '%s'", r.dynakube.ActiveGateAuthTokenSecret())
	}

	return r.createSecret(ctx, tokenSecretData)
}

func (r *Reconciler) createRandomToken(ctx context.Context) (string, error) {
	token, err := dttoken.New("")
	if err != nil {
		conditions.SetDynatraceApiError(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, err)

		return "", errors.WithStack(err)
	}
	return token.String(), nil
}

func (r *Reconciler) generateNodeConfigCollectorToken(ctx context.Context) (map[string][]byte, error) {
	tokenString, err := r.createRandomToken(ctx)
	if err != nil {
		conditions.SetDynatraceApiError(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, err)

		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		NodeConfigCollectorTokenName: []byte(tokenString),
	}, nil
}

func (r *Reconciler) reconcileNodeConfigCollectorTokenSecret(ctx context.Context) error {
	var secret corev1.Secret

	err := r.apiReader.Get(ctx,
		client.ObjectKey{Name: r.dynakube.NodeConfigCollectorTokenSecret(), Namespace: r.dynakube.Namespace},
		&secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("creating nodeConfigCollectorToken secret")

			return r.ensureNodeConfigCollectorTokenSecret(ctx)
		}

		conditions.SetKubeApiError(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, err)

		return errors.WithStack(err)
	}

	if isSecretOutdated(&secret) {
		log.Info("nodeConfigCollectorToken is outdated, creating new one")

		conditions.SetSecretOutdated(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, "secret is outdated, update in progress")

		if err := r.deleteSecret(ctx, &secret); err != nil {
			return errors.WithStack(err)
		}

		return r.ensureNodeConfigCollectorTokenSecret(ctx)
	}

	r.conditionSetSecretCreated(&secret) // update message once a day

	return nil
}

func (r *Reconciler) createSecret(ctx context.Context, secretData map[string][]byte) error {
	secretName := r.dynakube.NodeConfigCollectorTokenSecret()

	secret, err := secret.Create(r.dynakube,
		secret.NewNameModifier(secretName),
		secret.NewNamespaceModifier(r.dynakube.Namespace),
		secret.NewDataModifier(secretData))
	if err != nil {
		conditions.SetKubeApiError(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, err)

		return errors.WithStack(err)
	}

	err = r.client.Create(ctx, secret)
	if err != nil {
		conditions.SetKubeApiError(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, err)

		return errors.Errorf("failed to create secret '%s': %v", secretName, err)
	}

	r.conditionSetSecretCreated(secret)

	return nil
}

func (r *Reconciler) deleteSecret(ctx context.Context, secret *corev1.Secret) error {
	if err := r.client.Delete(ctx, secret); err != nil && !k8serrors.IsNotFound(err) {
		conditions.SetKubeApiError(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, err)

		return err
	}

	return nil
}

func isSecretOutdated(secret *corev1.Secret) bool {
	return secret.CreationTimestamp.Add(AuthTokenRotationInterval).Before(time.Now())
}

func (r *Reconciler) conditionSetSecretCreated(secret *corev1.Secret) {
	lifespan := time.Since(secret.CreationTimestamp.Time)
	days := strconv.Itoa(int(lifespan.Hours() / 24))
	tokenAllParts := strings.Split(string(secret.Data[NodeConfigCollectorTokenName]), ".")
	tokenPublicPart := strings.Join(tokenAllParts[:2], ".")

	setNodeConfigCollectorSecretCreated(r.dynakube.Conditions(), NodeConfigCollectorTokenSecretConditionType, "secret created "+days+" day(s) ago, token:"+tokenPublicPart)
}
