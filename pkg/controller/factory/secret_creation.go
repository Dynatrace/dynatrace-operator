package factory

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SecretManager struct {
	Client client.Client
	Scheme *runtime.Scheme
	Logger logr.Logger
	Secret *corev1.Secret
	Owner  metav1.Object
}

// CreateOrUpdateSecret creates a secret in case it does not exist or updates it if there are changes
func CreateOrUpdateSecret(secretManager *SecretManager) error {
	var cfg corev1.Secret
	err := secretManager.Client.Get(context.TODO(), client.ObjectKey{Name: secretManager.Secret.Name, Namespace: secretManager.Secret.Namespace}, &cfg)
	if k8serrors.IsNotFound(err) {
		secretManager.Logger.Info("Creating OneAgent config secret")
		// Set DynaKube instance as the owner and controller
		if err := controllerutil.SetControllerReference(secretManager.Owner, secretManager.Secret, secretManager.Scheme); err != nil {
			secretManager.Logger.Error(err, "error setting controller reference")
			return err
		}

		if err := secretManager.Client.Create(context.TODO(), secretManager.Secret); err != nil {
			return fmt.Errorf("failed to create secret %s: %w", secretManager.Secret.Name, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to query for secret %s: %w", secretManager.Secret.Name, err)
	}

	if !reflect.DeepEqual(secretManager.Secret.Data, cfg.Data) {
		secretManager.Logger.Info("Updating secret", "secret", secretManager.Secret.Name)
		cfg.Data = secretManager.Secret.Data
		if err := secretManager.Client.Update(context.TODO(), &cfg); err != nil {
			return fmt.Errorf("failed to update secret %s: %w", secretManager.Secret.Name, err)
		}
	}

	return nil
}
