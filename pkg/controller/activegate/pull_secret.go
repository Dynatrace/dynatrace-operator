package activegate

import (
	"context"
	"fmt"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/apis/dynatrace/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/builder"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/factory"
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-operator/pkg/dtclient"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ReconcileActiveGate) reconcilePullSecret(instance *dynatracev1alpha1.DynaKube, log logr.Logger, dtc dtclient.Client) error {
	var tkns corev1.Secret
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: parser.GetTokensName(instance), Namespace: instance.GetNamespace()}, &tkns); err != nil {
		return fmt.Errorf("failed to query tokens: %w", err)
	}
	pullSecretData, err := builder.GeneratePullSecretData(instance, dtc, &tkns)
	if err != nil {
		return fmt.Errorf("failed to generate pull secret data: %w", err)
	}
	err = factory.CreateOrUpdateSecret(r.client, r.client, r.scheme, instance, instance.GetNamespace(), pullSecretData, corev1.SecretTypeDockerConfigJson, log)
	if err != nil {
		return fmt.Errorf("failed to create or update secret: %w", err)
	}

	return nil
}
