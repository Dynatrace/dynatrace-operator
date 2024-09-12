package eec

import (
	"crypto/x509"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/extension/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/certificates"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	client    client.Client
	apiReader client.Reader

	dk *dynakube.DynaKube
}

type ReconcilerBuilder func(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler

var _ ReconcilerBuilder = NewReconciler

func NewReconciler(clt client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler {
	return &reconciler{
		client:    clt,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *reconciler) Reconcile(ctx context.Context) error {
	if !r.dk.PrometheusEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), extensionsControllerStatefulSetConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), extensionsControllerStatefulSetConditionType)

		sts, err := statefulset.Build(r.dk, dynakube.ExtensionsExecutionControllerStatefulsetName, corev1.Container{})
		if err != nil {
			log.Error(err, "could not build "+dynakube.ExtensionsExecutionControllerStatefulsetName+" during cleanup")

			return err
		}

		err = statefulset.Query(r.client, r.apiReader, log).Delete(ctx, sts)

		if err != nil {
			log.Error(err, "failed to clean up "+dynakube.ExtensionsExecutionControllerStatefulsetName+" statufulset")

			return nil
		}

		return nil
	}

	err := r.reconcileTlsSecret(ctx)
	if err != nil {
		return err
	}

	if r.dk.Status.ActiveGate.ConnectionInfoStatus.TenantUUID == "" {
		conditions.SetStatefulSetOutdated(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, dynakube.ExtensionsExecutionControllerStatefulsetName)

		return errors.New("tenantUUID unknown")
	}

	if r.dk.Status.KubeSystemUUID == "" {
		conditions.SetStatefulSetOutdated(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, dynakube.ExtensionsExecutionControllerStatefulsetName)

		return errors.New("kubeSystemUUID unknown")
	}

	return r.createOrUpdateStatefulset(ctx)
}

func (r *reconciler) reconcileTlsSecret(ctx context.Context) error {
	query := k8ssecret.Query(r.client, r.client, log)

	secret, err := query.Get(ctx, types.NamespacedName{
		Name:      getTlsSecretName(r.dk.Name),
		Namespace: r.dk.Namespace,
	})

	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if r.dk.Spec.Templates.ExtensionExecutionController.TlsRefName != "" {
		if secret != nil {
			return query.Delete(ctx, secret)
		}

		return nil
	}

	if k8serrors.IsNotFound(err) {
		err = r.createTlsSecret(ctx)
		if err != nil {
			return err
		}

		return nil
	}

	if secret != nil {
		err = r.reconcileTlsSecretExpiration(ctx, secret)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *reconciler) createTlsSecret(ctx context.Context) error {
	cert, err := certificates.New()
	cert.Cert.DNSNames = getCertificateAltNames(r.dk.Name)
	cert.Cert.KeyUsage = x509.KeyUsageKeyEncipherment | x509.KeyUsageDataEncipherment
	cert.Cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}

	if err != nil {
		return err
	}

	err = cert.SelfSign()
	if err != nil {
		return err
	}

	certPem, pkPem, err := cert.ToPEM()
	if err != nil {
		return err
	}

	coreLabels := k8slabels.NewCoreLabels(r.dk.Name, k8slabels.ExtensionComponentLabel)
	secretData := map[string][]byte{consts.TLSCrtDataName: certPem, consts.TLSKeyDataName: pkPem}

	secret, err := k8ssecret.Build(r.dk, getTlsSecretName(r.dk.Name), secretData, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		return err
	}

	secret.Type = corev1.SecretTypeTLS

	query := k8ssecret.Query(r.client, r.client, log)

	_, err = query.CreateOrUpdate(ctx, secret)
	if err != nil {
		return err
	}

	return nil
}

func (r *reconciler) reconcileTlsSecretExpiration(ctx context.Context, secret *corev1.Secret) error {
	// WIP
	return nil
}

func getCertificateAltNames(dkName string) []string {
	return []string{
		dkName + "-extensions-controller.dynatrace",
		dkName + "-extensions-controller.dynatrace.svc",
		dkName + "-extensions-controller.dynatrace.svc.cluster.local",
	}
}

func getTlsSecretName(dkName string) string {
	return dkName + consts.ExtensionsTlsSecretSuffix
}
