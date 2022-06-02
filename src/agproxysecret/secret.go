package agproxysecret

import (
	"context"
	"fmt"
	"net/url"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	agcapability "github.com/Dynatrace/dynatrace-operator/src/controllers/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	proxyHostField     = "host"
	proxyPortField     = "port"
	proxyUsernameField = "username"
	proxyPasswordField = "password"

	ProxySecretKey              = "proxy"
	activeGateProxySecretSuffix = "internal-proxy"
)

// ActiveGateProxySecretGenerator manages the ActiveGate proxy secret generation for the dynatrace namespace.
type ActiveGateProxySecretGenerator struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

func NewActiveGateProxySecretGenerator(client client.Client, apiReader client.Reader, ns string, logger logr.Logger) *ActiveGateProxySecretGenerator {
	return &ActiveGateProxySecretGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: ns,
		logger:    logger,
	}
}

func (agProxySecretGenerator *ActiveGateProxySecretGenerator) GenerateForDynakube(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	data, err := agProxySecretGenerator.createProxyMap(ctx, dynakube)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := kubeobjects.NewCoreLabels(dynakube.Name, kubeobjects.ActiveGateComponentLabel)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      BuildProxySecretName(),
			Namespace: agProxySecretGenerator.namespace,
			Labels:    coreLabels.BuildMatchLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
	query := kubeobjects.NewSecretQuery(ctx, agProxySecretGenerator.client, agProxySecretGenerator.apiReader, agProxySecretGenerator.logger)

	return errors.WithStack(query.CreateOrUpdate(*secret))
}

func (agProxySecretGenerator *ActiveGateProxySecretGenerator) EnsureDeleted(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) error {
	secretName := BuildProxySecretName()
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: dynakube.Namespace}}
	if err := agProxySecretGenerator.client.Delete(ctx, &secret); err != nil && !k8serrors.IsNotFound(err) {
		return err
	} else if err == nil {
		// If the secret is deleted the error is nil, otherwise err is notFound, then we should log nothing
		agProxySecretGenerator.logger.Info("removed secret", "namespace", dynakube.Namespace, "secret", secretName)
	}
	return nil
}

func BuildProxySecretName() string {
	return "dynatrace" + "-" + agcapability.MultiActiveGateName + "-" + activeGateProxySecretSuffix
}

func (agProxySecretGenerator *ActiveGateProxySecretGenerator) createProxyMap(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
	var err error
	proxyUrl := ""
	if dynakube.Spec.Proxy != nil && dynakube.Spec.Proxy.ValueFrom != "" {
		if proxyUrl, err = agProxySecretGenerator.proxyUrlFromUserSecret(ctx, dynakube); err != nil {
			return nil, err
		}
	} else if dynakube.Spec.Proxy != nil && len(dynakube.Spec.Proxy.Value) > 0 {
		proxyUrl = proxyUrlFromSpec(dynakube)
	} else {
		// the parsed-proxy secret is expected to exist and the entrypoint.sh script handles empty values properly
		return map[string][]byte{
			proxyHostField:     []byte(""),
			proxyPortField:     []byte(""),
			proxyUsernameField: []byte(""),
			proxyPasswordField: []byte(""),
		}, nil
	}

	host, port, username, password, err := parseProxyUrl(proxyUrl)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		proxyHostField:     []byte(host),
		proxyPortField:     []byte(port),
		proxyUsernameField: []byte(username),
		proxyPasswordField: []byte(password),
	}, nil
}

func proxyUrlFromSpec(dynakube *dynatracev1beta1.DynaKube) string {
	return dynakube.Spec.Proxy.Value
}

func (agProxySecretGenerator *ActiveGateProxySecretGenerator) proxyUrlFromUserSecret(ctx context.Context, dynakube *dynatracev1beta1.DynaKube) (string, error) {
	var proxySecret corev1.Secret
	if err := agProxySecretGenerator.client.Get(ctx, client.ObjectKey{Name: dynakube.Spec.Proxy.ValueFrom, Namespace: agProxySecretGenerator.namespace}, &proxySecret); err != nil {
		return "", errors.WithMessage(err, fmt.Sprintf("failed to query %s secret", dynakube.Spec.Proxy.ValueFrom))
	}

	proxy, ok := proxySecret.Data[ProxySecretKey]
	if !ok {
		return "", fmt.Errorf("invalid secret %s", dynakube.Spec.Proxy.ValueFrom)
	}
	return string(proxy), nil
}

func parseProxyUrl(proxy string) (host string, port string, username string, password string, err error) {
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return "", "", "", "", errors.New("could not parse proxy URL")
	}

	passwd, _ := proxyUrl.User.Password()
	return proxyUrl.Hostname(), proxyUrl.Port(), proxyUrl.User.Username(), passwd, nil
}
