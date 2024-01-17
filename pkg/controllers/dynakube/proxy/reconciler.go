package proxy

import (
	"context"
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ controllers.Reconciler = &Reconciler{}

// Reconciler manages the proxy secret generation for the dynatrace namespace.
type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	scheme    *runtime.Scheme
	dynakube  *dynatracev1beta1.DynaKube
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dynakube.NeedsActiveGateProxy() || r.dynakube.NeedsOneAgentProxy() {
		return r.generateForDynakube(ctx, r.dynakube)
	}

	return r.ensureDeleted(ctx, r.dynakube)
}

func NewReconciler(client client.Client, apiReader client.Reader, scheme *runtime.Scheme, dynakube *dynatracev1beta1.DynaKube) *Reconciler {
	return &Reconciler{
		client:    client,
		apiReader: apiReader,
		scheme:    scheme,
		dynakube:  dynakube,
	}
}

func (r *Reconciler) generateForDynakube(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	data, err := r.createProxyMap(ctx, dynakube)
	if err != nil {
		return errors.WithStack(err)
	}

	secret, err := k8ssecret.Create(r.scheme, r.dynakube,
		k8ssecret.NewNameModifier(BuildSecretName(dynakube.Name)),
		k8ssecret.NewNamespaceModifier(r.dynakube.Namespace),
		k8ssecret.NewTypeModifier(corev1.SecretTypeOpaque),
		k8ssecret.NewDataModifier(data))
	if err != nil {
		return errors.WithStack(err)
	}

	secretQuery := k8ssecret.NewQuery(ctx, r.client, r.apiReader, log)

	err = secretQuery.CreateOrUpdate(*secret)

	return errors.WithStack(err)
}

func (r *Reconciler) ensureDeleted(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	secretName := BuildSecretName(dynakube.Name)

	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: dynakube.Namespace}}
	if err := r.client.Delete(ctx, &secret); err != nil && !k8serrors.IsNotFound(err) {
		return err
	} else if err == nil {
		// If the secret is deleted the error is nil, otherwise err is notFound, then we should log nothing
		log.Info("removed secret", "namespace", dynakube.Namespace, "secret", secretName)
	}

	return nil
}

func (r *Reconciler) createProxyMap(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
	if !dynakube.HasProxy() {
		// the parsed-proxy secret is expected to exist and the entrypoint.sh script handles empty values properly
		return map[string][]byte{
			schemeField:   []byte(""),
			hostField:     []byte(""),
			portField:     []byte(""),
			usernameField: []byte(""),
			passwordField: []byte(""),
		}, nil
	}

	proxyUrl, err := dynakube.Proxy(ctx, r.apiReader)
	if err != nil {
		return nil, err
	}

	scheme, host, port, username, password, err := parseProxyUrl(proxyUrl)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		schemeField:   []byte(scheme),
		hostField:     []byte(host),
		portField:     []byte(port),
		usernameField: []byte(username),
		passwordField: []byte(password),
	}, nil
}

func parseProxyUrl(proxy string) (scheme, host, port, username, password string, err error) { //nolint:revive // maximum number of return results per function exceeded; max 3 but got 5
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return "", "", "", "", "", errors.New("could not parse proxy URL")
	}

	passwd, _ := proxyUrl.User.Password()

	return proxyUrl.Scheme, proxyUrl.Hostname(), proxyUrl.Port(), proxyUrl.User.Username(), passwd, nil
}

func BuildSecretName(dynakubeName string) string {
	return dynakubeName + "-" + consts.ProxySecretSuffix
}
