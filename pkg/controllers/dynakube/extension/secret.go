package extension

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	eecConsts "github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNoMigration = errors.New("no migration needed")

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	if migrationNeeded() {
		// remove condition and maybe old secret
	}

	reconcileAsBefore()
}

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	if !r.dk.Extensions().IsEnabled() {
		r.cleanupSecret(ctx)

		return nil
	}

	existingSecret, err := r.secrets.Get(ctx, client.ObjectKey{Name: r.getSecretName(), Namespace: r.dk.Namespace})
	if err != nil && !k8serrors.IsNotFound(err) {
		log.Info("failed to check existence of extension secret")
		conditions.SetKubeAPIError(r.dk.Conditions(), secretConditionType, err)

		return err
	}

	var newSecret *corev1.Secret
	if k8serrors.IsNotFound(err) {
		newSecret, err = r.newSecret()
	} else if r.isMigrationNeeded(existingSecret) {
		newSecret, err = r.migrateSecret(existingSecret)
	}
	_, err = r.secrets.CreateOrUpdate(ctx, newSecret)
	if err != nil {
		log.Info("failed to create/update extension secret")
		conditions.SetKubeAPIError(r.dk.Conditions(), secretConditionType, err)

		return err
	}

	conditions.SetSecretCreated(r.dk.Conditions(), secretConditionType, r.getSecretName())

	return nil
}

func (r *reconciler) buildSecret(eecToken []byte, datasourceToken []byte) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecConsts.TokenSecretKey:        eecToken,
		consts.DatasourceTokenSecretKey: datasourceToken,
	}

	return k8ssecret.Build(r.dk, r.getSecretName(), secretData)
}

func (r *reconciler) getSecretName() string {
	return r.dk.Extensions().GetTokenSecretName()
}

func (r *reconciler) newSecret() (*corev1.Secret, error) {
	newEecToken, err := dttoken.New(eecConsts.TokenSecretValuePrefix)
	if err != nil {
		log.Info("failed to generate eec token")
		conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, errors.Wrap(err, "error generating eec token"))

		return nil, err
	}

	newOtelcToken, err := dttoken.New(consts.DatasourceTokenSecretValuePrefix)
	if err != nil {
		log.Info("failed to generate otelc token")
		conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, errors.Wrap(err, "error generating otelc token"))

		return nil, err
	}

	newSecret, err := r.buildSecret([]byte(newEecToken.String()), []byte(newOtelcToken.String()))
	if err != nil {
		log.Info("failed to generate extension secret")
		conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, err)

		return nil, err
	}

	return newSecret, nil
}
func (r *reconciler) isMigrationNeeded(existingSecret *corev1.Secret) bool {
	if existingSecret == nil {
		return false
	}

	_, datasourceTokenExists := existingSecret.Data[consts.DatasourceTokenSecretKey]

	return !datasourceTokenExists

}

func (r *reconciler) migrateSecret(existingSecret *corev1.Secret) (*corev1.Secret, error) {
	const oTelcTokenKey = "otelc.token"

	if _, exists := existingSecret.Data[consts.DatasourceTokenSecretKey]; exists {
		// no migration needed, datasource token file name already correct
		return nil, ErrNoMigration
	}

	datasourceToken, exists := existingSecret.Data[oTelcTokenKey]
	if !exists {
		conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, errors.Errorf("error migrating datasource token, missing %s key", oTelcTokenKey))

		return nil, errors.New("missing otelc token in existing secret, cannot migrate")
	}

	eecToken, exists := existingSecret.Data[eecConsts.TokenSecretKey]
	if !exists {
		conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, errors.Errorf("error migrating EEC token, missing %s key", eecConsts.TokenSecretKey))

		return nil, errors.New("missing EEC token in existing secret, cannot migrate")
	}

	migratedSecret, err := r.buildSecret(eecToken, datasourceToken)
	if err != nil {
		conditions.SetSecretGenFailed(r.dk.Conditions(), secretConditionType, err)

		return nil, err
	}

	return migratedSecret, nil
}

func (r *reconciler) cleanupSecret(ctx context.Context) {
	if meta.FindStatusCondition(*r.dk.Conditions(), secretConditionType) == nil {
		return
	}
	defer meta.RemoveStatusCondition(r.dk.Conditions(), secretConditionType)

	secret, err := r.buildSecret([]byte{}, []byte{})
	if err != nil {
		log.Error(err, "failed to build empty extension secret during cleanup")

		return
	}

	err = r.secrets.Delete(ctx, secret)
	if err != nil {
		log.Error(err, "failed to clean up extension secret")
	}
}
