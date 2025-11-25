package proxy

import (
	"context"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8ssecret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ controllers.Reconciler = &Reconciler{}

var _ ReconcilerBuilder = NewReconciler

type ReconcilerBuilder func(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler

// Reconciler manages the proxy secret generation for the dynatrace namespace.
type Reconciler struct {
	client    client.Client
	apiReader client.Reader
	dk        *dynakube.DynaKube
}

func (r *Reconciler) Reconcile(ctx context.Context) error {
	if r.dk.NeedsActiveGateProxy() || r.dk.NeedsOneAgentProxy() {
		return r.generateForDynakube(ctx, r.dk)
	}

	return r.ensureDeleted(ctx, r.dk)
}

func NewReconciler(client client.Client, apiReader client.Reader, dk *dynakube.DynaKube) controllers.Reconciler {
	return &Reconciler{
		client:    client,
		apiReader: apiReader,
		dk:        dk,
	}
}

func (r *Reconciler) generateForDynakube(ctx context.Context, dk *dynakube.DynaKube) error {
	data, err := r.createProxyMap(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	secret, err := k8ssecret.Build(r.dk,
		BuildSecretName(dk.Name),
		data,
		k8ssecret.SetType(corev1.SecretTypeOpaque),
	)
	if err != nil {
		return errors.WithStack(err)
	}

	secrets := k8ssecret.Query(r.client, r.apiReader, log)

	_, err = secrets.CreateOrUpdate(ctx, secret)

	return errors.WithStack(err)
}

func (r *Reconciler) ensureDeleted(ctx context.Context, dk *dynakube.DynaKube) error {
	secretName := BuildSecretName(dk.Name)

	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: dk.Namespace}}
	if err := r.client.Delete(ctx, &secret); err != nil && !k8serrors.IsNotFound(err) {
		return err
	} else if err == nil {
		// If the secret is deleted the error is nil, otherwise err is notFound, then we should log nothing
		log.Info("removed secret", "namespace", dk.Namespace, "secret", secretName)
	}

	return nil
}

func (r *Reconciler) createProxyMap(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	if !dk.HasProxy() {
		dk.Status.ProxyURLHash = ""

		// the parsed-proxy secret is expected to exist and the entrypoint.sh script handles empty values properly
		return map[string][]byte{
			hostField:     []byte(""),
			portField:     []byte(""),
			usernameField: []byte(""),
			passwordField: []byte(""),
			schemeField:   []byte(""),
		}, nil
	}

	proxyURL, err := dk.Proxy(ctx, r.apiReader)
	if err != nil {
		return nil, err
	}

	dk.Status.ProxyURLHash, err = hasher.GenerateSecureHash(proxyURL)
	if err != nil {
		return nil, err
	}

	scheme, host, port, username, password, err := parseProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		hostField:     []byte(host),
		portField:     []byte(port),
		usernameField: []byte(username),
		passwordField: []byte(password),
		schemeField:   []byte(scheme),
	}, nil
}

func parseProxyURL(proxy string) (scheme, host, port, username, password string, err error) { //nolint:revive // maximum number of return results per function exceeded; max 3 but got 6
	if !strings.HasPrefix(strings.ToLower(proxy), "http://") && !strings.HasPrefix(strings.ToLower(proxy), "https://") {
		log.Info("proxy url has no scheme. The default 'http://' scheme used")

		proxy = "http://" + proxy
	}

	proxyURL, err := url.Parse(proxy)
	if err != nil {
		return "", "", "", "", "", errors.New("could not parse proxy URL")
	}

	passwd, _ := proxyURL.User.Password()

	return proxyURL.Scheme, proxyURL.Hostname(), proxyURL.Port(), proxyURL.User.Username(), passwd, nil
}

func BuildSecretName(dynakubeName string) string {
	return dynakubeName + "-" + consts.ProxySecretSuffix
}
