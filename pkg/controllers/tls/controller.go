// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package tls

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/eventfilter"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	k8sobject "github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects"
	"github.com/Dynatrace/dynatrace-operator/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// CABundleConfigMapName is the name of the ConfigMap holding the public part of the operator root CA.
	CABundleConfigMapName = "dynatrace-tls-ca-bundle"

	// CASecretName is the name of the Secret holding the operator root CA (certificate and key).
	CASecretName = "dynatrace-tls-ca"

	CACertKey = "ca.crt"
	CAKeyKey  = "ca.key"

	controllerName = "tls-ca-controller"
)

// ErrCANotFound is returned when the operator root CA Secret does not exist.
var ErrCANotFound = errors.New("operator root CA secret not found")

// Controller owns the operator root CA: it creates it if missing and regenerates it if invalid,
// independent of any DynaKube. It reacts to every change of the CA Secret and of the CA bundle ConfigMap.
type Controller struct {
	client    client.Client
	namespace string
}

func Add(mgr manager.Manager, namespace string) error {
	return NewController(mgr.GetClient(), namespace).SetupWithManager(mgr)
}

func NewController(clt client.Client, namespace string) *Controller {
	return &Controller{
		client:    clt,
		namespace: namespace,
	}
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	req := reconcile.Request{Name: CASecretName, Namespace: c.namespace}

	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&corev1.Secret{}, builder.WithPredicates(eventfilter.ForObjectNameAndNamespace(CASecretName, c.namespace))).
		Watches(&corev1.ConfigMap{},
			handler.EnqueueRequestsFromMapFunc(func(context.Context, client.Object) []reconcile.Request {
				return []reconcile.Request{req}
			}),
			builder.WithPredicates(eventfilter.ForObjectNameAndNamespace(CABundleConfigMapName, c.namespace)),
		).
		// start-up trigger, so the CA is created even if neither the Secret nor the ConfigMap exist
		WatchesRawSource(source.Func(func(_ context.Context, q workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
			q.Add(req)

			return nil
		})).
		Complete(c)
}

func (c *Controller) Reconcile(ctx context.Context, _ reconcile.Request) (reconcile.Result, error) {
	caSecret, err := c.ensureCA(ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, c.ensureBundle(ctx, caSecret)
}

// ensureCA makes sure the CA Secret holds a valid CA and returns the Secret. A valid CA Secret is never written,
// a missing or an invalid one is (re)created. The Secret is owned by the operator Deployment, so it is removed with the Operator.
func (c *Controller) ensureCA(ctx context.Context) (*corev1.Secret, error) {
	log := logd.FromContext(ctx)

	owner, err := c.getOperatorDeployment(ctx)
	if err != nil {
		return nil, err
	}

	secret := &corev1.Secret{Name: CASecretName, Namespace: c.namespace}

	err = k8sobject.RetryCreateOrUpdate(ctx, c.client, secret, func() error {
		// after a recreate the object still holds the data of the deleted Secret, which must not be validated again
		if secret.ResourceVersion == "" {
			secret.Data = nil
		}

		// the type is immutable, so it is only set on creation
		if secret.ResourceVersion == "" {
			secret.Type = corev1.SecretTypeOpaque
		} else if secret.Type != corev1.SecretTypeOpaque {
			return k8sobject.ErrRecreate
		}

		k8slabel.New(k8slabel.OperatorComponentLabel, version.AppName, "").MergeInto(secret)

		if owner != nil {
			err := controllerutil.SetControllerReference(owner, secret, c.client.Scheme())
			if err != nil {
				return err
			}
		}

		_, err := certificates.ParseCA(secret.Data[CACertKey], secret.Data[CAKeyKey], time.Now())
		if err == nil {
			return nil
		}

		switch {
		case len(secret.Data) == 0:
			log.Info("creating operator root CA", "name", CASecretName, "namespace", c.namespace)
		case errors.Is(err, certificates.ErrInvalidCA):
			log.Info("operator root CA is invalid, regenerating it", "reason", err.Error(), "name", CASecretName, "namespace", c.namespace)

			// Note: This case is rather unlikely. As the CA would only go invalid after 10 years and the Secret is `immutable`.
			// The secret being immutable is also the reason to need this recreate
			return k8sobject.ErrRecreate
		default:
			return err
		}

		ca, err := certificates.NewCA(time.Now())
		if err != nil {
			return err
		}

		secret.Data = map[string][]byte{
			CACertKey: ca.CertPEM,
			CAKeyKey:  ca.KeyPEM,
		}

		secret.Immutable = new(true)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return secret, nil
}

// getOperatorDeployment returns the Deployment of the Operator, or nil if it does not exist, for example if the Operator runs locally.
func (c *Controller) getOperatorDeployment(ctx context.Context) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{}

	err := c.client.Get(ctx, types.NamespacedName{Name: version.AppName, Namespace: c.namespace}, deployment)
	if k8serrors.IsNotFound(err) {
		logd.FromContext(ctx).Info("operator deployment not found, the operator root CA will not be owned by it", "name", version.AppName, "namespace", c.namespace)

		return nil, nil //nolint:nilnil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get operator deployment: %w", err)
	}

	return deployment, nil
}

// ensureBundle makes sure the bundle ConfigMap holds the public part of the CA. The ConfigMap is immutable,
// so it is recreated if the CA changed. It is owned by the CA Secret, so it is removed with it.
func (c *Controller) ensureBundle(ctx context.Context, caSecret *corev1.Secret) error {
	configMap := &corev1.ConfigMap{Name: CABundleConfigMapName, Namespace: c.namespace}
	data := map[string]string{CACertKey: string(caSecret.Data[CACertKey])}

	return k8sobject.RetryCreateOrUpdate(ctx, c.client, configMap, func() error {
		if configMap.ResourceVersion != "" && !maps.Equal(configMap.Data, data) {
			return k8sobject.ErrRecreate
		}

		k8slabel.New(k8slabel.OperatorComponentLabel, version.AppName, "").MergeInto(configMap)

		configMap.Data = data
		configMap.Immutable = new(true)

		// not a controller reference, as that would block the deletion of the Secret, which needs additional permissions (secrets/finalizers)
		return controllerutil.SetOwnerReference(caSecret, configMap, c.client.Scheme())
	})
}

// loadCA reads and validates the operator root CA from the operator namespace, it never writes anything.
// It returns ErrCANotFound (wrapped) if the Secret does not exist and certificates.ErrInvalidCA (wrapped) if it is not usable.
func loadCA(ctx context.Context, clt client.Client, namespace string, now time.Time) (*certificates.CA, error) {
	secret := &corev1.Secret{}

	err := clt.Get(ctx, types.NamespacedName{Name: CASecretName, Namespace: namespace}, secret)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: %s/%s", ErrCANotFound, namespace, CASecretName)
		}

		return nil, fmt.Errorf("failed to get operator root CA secret: %w", err)
	}

	return certificates.ParseCA(secret.Data[CACertKey], secret.Data[CAKeyKey], now)
}
