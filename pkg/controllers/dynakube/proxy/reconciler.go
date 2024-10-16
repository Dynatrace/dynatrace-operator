package proxy

import (
	"context"
	"net/url"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/consts"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
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

	secretQuery := k8ssecret.Query(r.client, r.apiReader, log)

	_, err = secretQuery.CreateOrUpdate(ctx, secret)

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
		// the parsed-proxy secret is expected to exist and the entrypoint.sh script handles empty values properly
		return map[string][]byte{
			hostField:     []byte(""),
			portField:     []byte(""),
			usernameField: []byte(""),
			passwordField: []byte(""),
			schemeField:   []byte(""),
		}, nil
	}

	proxyUrl, err := dk.Proxy(ctx, r.apiReader)
	if err != nil {
		return nil, err
	}

	scheme, host, port, username, password, err := parseProxyUrl(proxyUrl)
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

func parseProxyUrl(proxy string) (scheme, host, port, username, password string, err error) { //nolint:revive // maximum number of return results per function exceeded; max 3 but got 6
	if !strings.HasPrefix(strings.ToLower(proxy), "http://") && !strings.HasPrefix(strings.ToLower(proxy), "https://") {
		log.Info("proxy url has no scheme. The default 'http://' scheme used")

		proxy = "http://" + proxy
	}

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
