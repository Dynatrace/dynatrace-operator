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
	parser "github.com/Dynatrace/dynatrace-activegate-operator/pkg/controller/parser"
	"github.com/Dynatrace/dynatrace-activegate-operator/pkg/dtclient"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-activegate-operator/pkg/apis/dynatrace/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

type options struct {
	Opts []dtclient.Option
}

// BuildDynatraceClient creates a new Dynatrace client using the settings configured on the given instance.
func BuildDynatraceClient(rtc client.Client, instance *dynatracev1alpha1.ActiveGate, secret *corev1.Secret) (dtclient.Client, error) {
	if instance == nil {
		return nil, fmt.Errorf("could not build dynatrace client: instance is nil")
	}
	namespace := instance.GetNamespace()
	spec := instance.Spec
	tokens, err := parser.NewTokens(secret)
	if err != nil {
		return nil, err
	}

	opts := newOptions()
	opts.appendCertCheck(&spec)

	err = opts.appendProxySettings(rtc, &spec, namespace)
	if err != nil {
		return nil, err
	}

	err = opts.appendTrustedCerts(rtc, &spec, namespace)
	if err != nil {
		return nil, err
	}

	return dtclient.NewClient(spec.APIURL, tokens.ApiToken, tokens.PaasToken, opts.Opts...)
}

func newOptions() *options {
	return &options{
		Opts: []dtclient.Option{},
	}
}

func (opts *options) appendCertCheck(spec *dynatracev1alpha1.ActiveGateSpec) {
	if spec.SkipCertCheck {
		opts.Opts = append(opts.Opts, dtclient.SkipCertificateValidation(true))
	}
}

func (opts *options) appendProxySettings(rtc client.Client, spec *dynatracev1alpha1.ActiveGateSpec, namespace string) error {
	if p := spec.Proxy; p != nil {
		if p.ValueFrom != "" {
			proxySecret := &corev1.Secret{}
			err := rtc.Get(context.TODO(), client.ObjectKey{Name: p.ValueFrom, Namespace: namespace}, proxySecret)
			if err != nil {
				return fmt.Errorf("failed to get proxy secret: %w", err)
			}

			proxyURL, err := parser.ExtractToken(proxySecret, Proxy)
			if err != nil {
				return fmt.Errorf("failed to extract proxy secret field: %w", err)
			}
			opts.Opts = append(opts.Opts, dtclient.Proxy(proxyURL))
		} else if p.Value != "" {
			opts.Opts = append(opts.Opts, dtclient.Proxy(p.Value))
		}
	}
	return nil
}

func (opts *options) appendTrustedCerts(rtc client.Client, spec *dynatracev1alpha1.ActiveGateSpec, namespace string) error {
	if spec.TrustedCAs != "" {
		certs := &corev1.ConfigMap{}
		if err := rtc.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: spec.TrustedCAs}, certs); err != nil {
			return fmt.Errorf("failed to get certificate configmap: %w", err)
		}
		if certs.Data["certs"] == "" {
			return fmt.Errorf("failed to extract certificate configmap field: missing field certs")
		}
		opts.Opts = append(opts.Opts, dtclient.Certs([]byte(certs.Data["certs"])))
	}
	return nil
}

const (
	Proxy = "proxy"
)
