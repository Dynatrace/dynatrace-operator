package initgeneration

import (
	"bytes"
	"context"
	"crypto/sha1"
	_ "embed"
	"fmt"
	"text/template"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	notMappedIM = "-"
)

var (
	//go:embed init.sh.tmpl
	scriptContent string
	scriptTmpl    = template.Must(template.New("initScript").Parse(scriptContent))
)

type InitGenerator struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

type script struct {
	ApiUrl        string
	SkipCertCheck bool
	PaaSToken     string
	Proxy         string
	TrustedCAs    []byte
	ClusterID     string
	TenantUUID    string
	IMNodes       map[string]string
}

func NewInitGenerator(client client.Client, apiReader client.Reader, ns string, logger logr.Logger) *InitGenerator {
	return &InitGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: ns,
		logger:    logger,
	}
}
func (g *InitGenerator) GenerateForNamespace(ctx context.Context, dkName, targetNs string) error {
	g.logger.Info("Reconciling namespace init secret for", "namespace", targetNs)
	var dk dynatracev1alpha1.DynaKube
	if err := g.client.Get(context.TODO(), client.ObjectKey{Name: dkName, Namespace: g.namespace}, &dk); err != nil {
		return err
	}
	data, err := g.generate(ctx, &dk)
	if err != nil {
		return err
	}
	return kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, webhook.SecretConfigName, targetNs, data, corev1.SecretTypeOpaque, g.logger)
}

func (g *InitGenerator) GenerateForDynakube(ctx context.Context, dk *dynatracev1alpha1.DynaKube) (bool, error) {
	g.logger.Info("Reconciling namespace init secret for", "dynakube", dk.Name)
	data, err := g.generate(ctx, dk)
	if err != nil {
		return false, err
	}
	hash, err := createHashForInitSecret(data)
	if err != nil {
		return false, err
	}
	if dk.Status.LastInitSecretHash == hash {
		g.logger.Info("No change in the init secret, no need to update", "dynakube", dk.Name)
		return false, nil
	}

	nsList, err := mapper.GetNamespacesForDynakube(ctx, mapper.CodeModulesAnnotation, g.apiReader, dk.Name)
	if err != nil {
		return false, err
	}
	for _, targetNs := range nsList {
		g.logger.Info("Updating init secret from dynakube for", "namespace", targetNs)
		if err = kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, webhook.SecretConfigName, targetNs.Name, data, corev1.SecretTypeOpaque, g.logger); err != nil {
			return false, err
		}
	}
	g.logger.Info("Done updating init secrets")
	dk.Status.LastInitSecretHash = hash
	return true, nil
}

func (g *InitGenerator) generate(ctx context.Context, dk *dynatracev1alpha1.DynaKube) (map[string][]byte, error) {
	kubeSystemUID, err := kubesystem.GetUID(g.apiReader)
	if err != nil {
		return nil, err
	}

	infraMonitoringNodes, err := g.getInfraMonitoringNodes(dk)
	if err != nil {
		return nil, err
	}

	script, err := g.prepareScriptForDynaKube(dk, kubeSystemUID, infraMonitoringNodes)
	if err != nil {
		return nil, err
	}

	data, err := script.generate()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (g *InitGenerator) prepareScriptForDynaKube(dk *dynatracev1alpha1.DynaKube, kubeSystemUID types.UID, infraMonitoringNodes map[string]string) (*script, error) {
	var tokens corev1.Secret
	if err := g.client.Get(context.TODO(), client.ObjectKey{Name: dk.Tokens(), Namespace: g.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	var proxy string
	if dk.Spec.Proxy != nil {
		if dk.Spec.Proxy.ValueFrom != "" {
			var ps corev1.Secret
			if err := g.client.Get(context.TODO(), client.ObjectKey{Name: dk.Spec.Proxy.ValueFrom, Namespace: g.namespace}, &ps); err != nil {
				return nil, fmt.Errorf("failed to query proxy: %w", err)
			}
			proxy = string(ps.Data["proxy"])
		} else if dk.Spec.Proxy.Value != "" {
			proxy = dk.Spec.Proxy.Value
		}
	}

	var trustedCAs []byte
	if dk.Spec.TrustedCAs != "" {
		var cam corev1.ConfigMap
		if err := g.client.Get(context.TODO(), client.ObjectKey{Name: dk.Spec.TrustedCAs, Namespace: g.namespace}, &cam); err != nil {
			return nil, fmt.Errorf("failed to query ca: %w", err)
		}
		trustedCAs = []byte(cam.Data["certs"])
	}

	return &script{
		ApiUrl:        dk.Spec.APIURL,
		SkipCertCheck: dk.Spec.SkipCertCheck,
		PaaSToken:     string(tokens.Data[dtclient.DynatracePaasToken]),
		Proxy:         proxy,
		TrustedCAs:    trustedCAs,
		ClusterID:     string(kubeSystemUID),
		TenantUUID:    dk.Status.ConnectionInfo.TenantUUID,
		IMNodes:       infraMonitoringNodes,
	}, nil
}

func (g *InitGenerator) getInfraMonitoringNodes(dk *dynatracev1alpha1.DynaKube) (map[string]string, error) {
	var dks dynatracev1alpha1.DynaKubeList
	if err := g.client.List(context.TODO(), &dks, client.InNamespace(g.namespace)); err != nil {
		return nil, errors.WithMessage(err, "failed to query DynaKubeList")
	}

	imNodes := map[string]string{}
	for i := range dks.Items {
		status := &dks.Items[i].Status
		if dk != nil && dk.Name == dks.Items[i].Name {
			status = &dk.Status
		}
		if dks.Items[i].Spec.InfraMonitoring.Enabled {
			tenantUUID := notMappedIM
			if status.ConnectionInfo.TenantUUID != "" {
				tenantUUID = status.ConnectionInfo.TenantUUID
			}
			for key := range status.OneAgent.Instances {
				if key != "" {
					imNodes[key] = tenantUUID
				}
			}
		}
	}

	return imNodes, nil
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

func createHashForInitSecret(initSecret map[string][]byte) (string, error) {
	bytes := []byte{}
	for _, part := range initSecret {
		bytes = append(bytes, part...)
	}
	h := sha1.New()
	if _, err := h.Write(bytes); err != nil {
		return "", err
	}
	bs := h.Sum(nil)

	return fmt.Sprintf("%x", bs), nil
}
