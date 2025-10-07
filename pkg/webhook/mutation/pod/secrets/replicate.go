package secrets

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/bootstrapperconfig"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EnsureReplicated ensures that a target secret exists in the given namespace.
// If the target secret does not exist, it tries to replicate it from the provided source secret name
// Returns nil on success (whether it already existed or was replicated) or an error if replication fails.
func EnsureReplicated(mutationRequest *dtwebhook.MutationRequest, kubeClient client.Client, apiReader client.Reader, sourceSecretName, targetSecretName string, logger logd.Logger) error {
	var initSecret corev1.Secret

	secretObjectKey := client.ObjectKey{Name: targetSecretName, Namespace: mutationRequest.Namespace.Name}

	err := apiReader.Get(mutationRequest.Context, secretObjectKey, &initSecret)
	if k8serrors.IsNotFound(err) {
		logger.Info(targetSecretName+" is not available, trying to replicate", "pod", mutationRequest.PodName())

		return bootstrapperconfig.Replicate(mutationRequest.Context, mutationRequest.DynaKube, secret.Query(kubeClient, apiReader, logger), sourceSecretName, targetSecretName, mutationRequest.Namespace.Name)
	}

	return nil
}
