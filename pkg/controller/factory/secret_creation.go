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

// CreateOrUpdateSecretIfNotExists creates a secret in case it does not exist or updates it if there are changes
func CreateOrUpdateSecretIfNotExists(c client.Client, r client.Reader, scheme *runtime.Scheme, owner metav1.Object, targetNS string, data map[string][]byte, secretType corev1.SecretType, log logr.Logger) error {
	secretName := owner.GetName() + "-pull-secret"
	var cfg corev1.Secret
	err := r.Get(context.TODO(), client.ObjectKey{Name: secretName, Namespace: targetNS}, &cfg)
	if k8serrors.IsNotFound(err) {
		log.Info("Creating OneAgent config secret")
		if err := c.Create(context.TODO(), &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: targetNS,
			},
			Type: secretType,
			Data: data,
		}); err != nil {
			return fmt.Errorf("failed to create secret %s: %w", secretName, err)
		}
		return nil
	}

	// Set DynaKube instance as the owner and controller
	if err := controllerutil.SetControllerReference(owner, &cfg, scheme); err != nil {
		log.Error(err, "error setting controller reference")
		return err
	}

	if err != nil {
		return fmt.Errorf("failed to query for secret %s: %w", secretName, err)
	}

	if !reflect.DeepEqual(data, cfg.Data) {
		log.Info(fmt.Sprintf("Updating secret %s", secretName))
		cfg.Data = data
		if err := c.Update(context.TODO(), &cfg); err != nil {
			return fmt.Errorf("failed to update secret %s: %w", secretName, err)
		}
	}

	return nil
}
