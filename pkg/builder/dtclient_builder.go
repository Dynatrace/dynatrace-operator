package builder

/**
The following functions have been copied from dynatrace-oneagent-operator
and are candidates to be made into a library:

* BuildDynatraceClient
* verifySecret
* getTokensName

*/

import (
	"context"
	"fmt"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	parser "github.com/Dynatrace/dynatrace-activegate-operator/pkg/parser"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// BuildDynatraceClient creates a new Dynatrace client using the settings configured on the given instance.
func BuildDynatraceClient(rtc client.Client, instance *dynatracev1alpha1.ActiveGate) (dtclient.Client, error) {
	ns := instance.GetNamespace()
	spec := instance.Spec

	secret := &corev1.Secret{}
	err := rtc.Get(context.TODO(), client.ObjectKey{Name: parser.GetTokensName(instance), Namespace: ns}, secret)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	tokens, err := parser.NewTokens(secret)
	if err != nil {
		return nil, err
	}

	// initialize dynatrace client
	var opts []dtclient.Option
	if spec.SkipCertCheck {
		opts = append(opts, dtclient.SkipCertificateValidation(true))
	}

	if p := spec.Proxy; p != nil {
		if p.ValueFrom != "" {
			proxySecret := &corev1.Secret{}
			err := rtc.Get(context.TODO(), client.ObjectKey{Name: p.ValueFrom, Namespace: ns}, proxySecret)
			if err != nil {
				return nil, fmt.Errorf("failed to get proxy secret: %w", err)
			}

			proxyURL, err := parser.ExtractToken(proxySecret, PROXY)
			if err != nil {
				return nil, fmt.Errorf("failed to extract proxy secret field: %w", err)
			}
			opts = append(opts, dtclient.Proxy(proxyURL))
		} else if p.Value != "" {
			opts = append(opts, dtclient.Proxy(p.Value))
		}
	}

	if spec.TrustedCAs != "" {
		certs := &corev1.ConfigMap{}
		if err := rtc.Get(context.TODO(), client.ObjectKey{Namespace: ns, Name: spec.TrustedCAs}, certs); err != nil {
			return nil, fmt.Errorf("failed to get certificate configmap: %w", err)
		}
		if certs.Data["certs"] == "" {
			return nil, fmt.Errorf("failed to extract certificate configmap field: missing field certs")
		}
		opts = append(opts, dtclient.Certs([]byte(certs.Data["certs"])))
	}

	//apiToken, err := parser.ExtractToken(secret, DynatraceApiToken)
	//if err != nil {
	//	return nil, err
	//}
	//
	//paasToken, err := parser.ExtractToken(secret, DynatracePaasToken)
	//if err != nil {
	//	return nil, err
	//}

	return dtclient.NewClient(spec.APIURL, tokens.ApiToken, tokens.PaasToken, opts...)
}

const (
	PROXY = "proxy"
)
