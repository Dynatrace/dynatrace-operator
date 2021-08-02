package namespace

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"text/template"
)

//go:embed init.sh.tmpl
var scriptContent string

var scriptTmpl = template.Must(template.New("initScript").Parse(scriptContent))

type initGenerator struct {
	c  client.Client
	dk *dynatracev1alpha1.DynaKube
	ns *corev1.Namespace
}

func (ig initGenerator) newScript(ctx context.Context) error {
	tokenName := ig.dk.Tokens()
	if !ig.dk.Spec.CodeModules.Enabled {
		_ = ensureSecretDeleted(ig.c, tokenName, ig.ns.Name)
		return nil // TODO: return someting
	}

	imNodes, err := ig.getIMNodes(ctx)
	if err != nil {
		return err
	}

	paasToken, err := ig.getPAASToken(ctx)
	if err != nil {
		return err
	}
	clusterId, err := ig.getClusterId(ctx)
	if err != nil {
		return err
	}

	proxy, err := ig.getProxy(ctx)
	if err != nil {
		return err
	}

	trustedCAs, err := ig.getTrustedCAs(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (ig initGenerator) getIMNodes(ctx context.Context) (map[string]string, error) {
	var ims dynatracev1alpha1.DynaKubeList
	if err := ig.c.List(ctx, &ims, client.InNamespace(ig.ns.Name)); err != nil {
		return nil, errors.WithMessage(err, "failed to query DynaKubeList")
	}

	imNodes := map[string]string{}
	for i := range ims.Items {
		if s := &ims.Items[i].Status; s.ConnectionInfo.TenantUUID != "" && ims.Items[i].Spec.InfraMonitoring.Enabled {
			for key := range s.OneAgent.Instances {
				if key != "" {
					imNodes[key] = s.ConnectionInfo.TenantUUID
				}
			}
		}
	}
	return imNodes, nil
}

func (ig initGenerator) getPAASToken(ctx context.Context) (string, error) {
	var tokens corev1.Secret
	if err := ig.c.Get(ctx, client.ObjectKey{Name: ig.dk.Tokens(), Namespace: ig.ns.Name}, &tokens); err != nil {
		return "", errors.WithMessage(err, "failed to query tokens")
	}
	return string(tokens.Data[utils.DynatracePaasToken]), nil
}

func (ig initGenerator) getClusterId(ctx context.Context) (string, error) {
	var kubeSystemNS corev1.Namespace
	if err := ig.c.Get(ctx, client.ObjectKey{Name: "kube-system"}, &kubeSystemNS); err != nil {
		return "", fmt.Errorf("failed to query for cluster ID: %w", err)
	}
	return string(kubeSystemNS.UID), nil
}

func (ig initGenerator) getProxy(ctx context.Context) (string, error) {
	var proxy string
	if ig.dk.Spec.Proxy != nil {
		if ig.dk.Spec.Proxy.ValueFrom != "" {
			var ps corev1.Secret
			if err := ig.c.Get(ctx, client.ObjectKey{Name: ig.dk.Spec.Proxy.ValueFrom, Namespace: ig.ns.Name}, &ps); err != nil {
				return "", fmt.Errorf("failed to query proxy: %w", err)
			}
			proxy = string(ps.Data["proxy"])
		} else if ig.dk.Spec.Proxy.Value != "" {
			proxy = ig.dk.Spec.Proxy.Value
		}
	}
	return proxy, nil
}

func (ig initGenerator) getTrustedCAs(ctx context.Context) ([]byte, error) {
	var trustedCAs []byte
	if ig.dk.Spec.TrustedCAs != "" {
		var cam corev1.ConfigMap
		if err := ig.c.Get(ctx, client.ObjectKey{Name: ig.dk.Spec.TrustedCAs, Namespace: ig.ns.Namespace}, &cam); err != nil {
			return nil, fmt.Errorf("failed to query ca: %w", err)
		}
		trustedCAs = []byte(cam.Data["certs"])
	}
	return trustedCAs, nil
}

func (s *script) generate() (map[string][]byte, error) {
	var buf bytes.Buffer

	if err := scriptTmpl.Execute(&buf, s); err != nil {
		return nil, err
	}

	data := map[string][]byte{
		"init.sh": buf.Bytes(),
	}

	if s.TrustedCAs != nil {
		data["ca.pem"] = s.TrustedCAs
	}

	if s.Proxy != "" {
		data["proxy"] = []byte(s.Proxy)
	}

	return data, nil
}

func ensureSecretDeleted(c client.Client, name string, ns string) error {
	secret := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
	if err := c.Delete(context.TODO(), &secret); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	return nil
}
