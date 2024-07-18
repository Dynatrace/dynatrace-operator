package extension

import (
	"context"
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/dttoken"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *reconciler) reconcileSecret(ctx context.Context) error {
	query := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

	if !r.dk.PrometheusEnabled() {
		return r.reconcileSecretDeleted(query)
	}

	_, err := query.Get(client.ObjectKey{Name: r.getSecretName(), Namespace: r.dk.Namespace})
	if err != nil && !errors.IsNotFound(err) {
		conditions.SetSecretCreatedFailed(r.dk.Conditions(), secretConditionType, fmt.Sprintf(secretCreatedMessageFailure, err))

		return err
	}

	if errors.IsNotFound(err) {
		newEecToken, err := dttoken.New(eecTokenSecretValuePrefix)
		if err != nil {
			conditions.SetSecretCreatedFailed(r.dk.Conditions(), secretConditionType, fmt.Sprintf(secretCreatedMessageFailure, err))

			return err
		}

		newSecret, err := r.buildSecret(*newEecToken)
		if err != nil {
			conditions.SetSecretCreatedFailed(r.dk.Conditions(), secretConditionType, fmt.Sprintf(secretCreatedMessageFailure, err))

			return err
		}

		err = query.CreateOrUpdate(*newSecret)
		if err != nil {
			conditions.SetSecretCreatedFailed(r.dk.Conditions(), secretConditionType, fmt.Sprintf(secretCreatedMessageFailure, err))

			return err
		}
	}

	conditions.SetSecretCreated(r.dk.Conditions(), secretConditionType, r.getSecretName())

	return nil
}

func (r *reconciler) reconcileSecretDeleted(query k8ssecret.Query) error {
	_, err := query.Get(client.ObjectKey{Name: r.getSecretName(), Namespace: r.dk.Namespace})
	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "failed reconciling deletion of "+r.getSecretName())

		return err
	}

	if !errors.IsNotFound(err) {
		err := query.Delete(r.getSecretName(), r.dk.Namespace)
		if err != nil {
			return err
		}
	}

	conditions.RemoveSecretCreated(r.dk.Conditions(), secretConditionType)

	return nil
}

func (r *reconciler) buildSecret(token dttoken.Token) (*corev1.Secret, error) {
	secretData := map[string][]byte{
		eecTokenSecretKey: []byte(token.String()),
	}

	return k8ssecret.Create(r.dk, k8ssecret.NewNameModifier(r.getSecretName()), k8ssecret.NewNamespaceModifier(r.dk.GetNamespace()), k8ssecret.NewDataModifier(secretData))
}

func (r *reconciler) getSecretName() string {
	return r.dk.Name + secretSuffix
}
