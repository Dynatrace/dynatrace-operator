package initgeneration

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	"github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	notMappedIM              = "-"
	trustedCASecretField     = "certs"
	proxyInitSecretField     = "proxy"
	trustedCAInitSecretField = "ca.pem"
	initScriptSecretField    = "init.sh"
)

var (
	//go:embed init.sh.tmpl
	scriptContent string
	scriptTmpl    = template.Must(template.New("initScript").Parse(scriptContent))
)

// InitGenerator manages the init secret generation for the user namespaces.
type InitGenerator struct {
	client    client.Client
	apiReader client.Reader
	logger    logr.Logger
	namespace string
}

type nodeInfo struct {
	nodes   []corev1.Node
	imNodes map[string]string
}

// script holds all the values to be passed to the init script template.
type script struct {
	ApiUrl        string
	SkipCertCheck bool
	PaaSToken     string
	Proxy         string
	TrustedCAs    []byte
	ClusterID     string
	TenantUUID    string
	IMNodes       map[string]string
	HostGroup     string
}

func NewInitGenerator(client client.Client, apiReader client.Reader, ns string, logger logr.Logger) *InitGenerator {
	return &InitGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: ns,
		logger:    logger,
	}
}

// GenerateForNamespace creates the init secret for namespace while only having the name of the corresponding dynakube
// Used by the podInjection webhook in case the namespace lacks the init secret.
func (g *InitGenerator) GenerateForNamespace(ctx context.Context, dkName, targetNs string) error {
	g.logger.Info("Reconciling namespace init secret for", "namespace", targetNs)
	var dk dynatracev1beta1.DynaKube
	if err := g.client.Get(context.TODO(), client.ObjectKey{Name: dkName, Namespace: g.namespace}, &dk); err != nil {
		return err
	}
	data, err := g.generate(ctx, &dk)
	if err != nil {
		return err
	}
	return kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, webhook.SecretConfigName, targetNs, data, corev1.SecretTypeOpaque, g.logger)
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *InitGenerator) GenerateForDynakube(ctx context.Context, dk *dynatracev1beta1.DynaKube) (bool, error) {
	g.logger.Info("Reconciling namespace init secret for", "dynakube", dk.Name)
	data, err := g.generate(ctx, dk)
	if err != nil {
		return false, err
	}

	nsList, err := mapper.GetNamespacesForDynakube(ctx, g.apiReader, dk.Name)
	if err != nil {
		return false, err
	}
	for _, targetNs := range nsList {
		if err = kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, webhook.SecretConfigName, targetNs.Name, data, corev1.SecretTypeOpaque, g.logger); err != nil {
			return false, err
		}
	}
	g.logger.Info("Done updating init secrets")
	return true, nil
}

// generate gets the necessary info the create the init secret data
func (g *InitGenerator) generate(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
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

func (g *InitGenerator) prepareScriptForDynaKube(dk *dynatracev1beta1.DynaKube, kubeSystemUID types.UID, infraMonitoringNodes map[string]string) (*script, error) {
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
			proxy = string(ps.Data[proxyInitSecretField])
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
		trustedCAs = []byte(cam.Data[trustedCASecretField])
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
		HostGroup:     getHostGroup(dk),
	}, nil
}

func getHostGroup(dk *dynatracev1beta1.DynaKube) string {
	var hostGroup string
	if dk.CloudNativeFullstackMode() && dk.Spec.OneAgent.CloudNativeFullStack.Args != nil {
		for _, arg := range dk.Spec.OneAgent.CloudNativeFullStack.Args {
			split := strings.Split(arg, "=")
			if len(split) != 2 {
				continue
			}
			key := split[0]
			value := split[1]
			if key == "--set-host-group" {
				hostGroup = value
				break
			}
		}
	}
	return hostGroup
}

// getInfraMonitoringNodes creates a mapping between all the nodes and the tenantUID for the infra-monitoring dynakube on that node.
// Possible mappings:
// - mapped: there is a infra-monitoring agent on the node, and the dynakube has the tenantUID set => user processes will be grouped to the hosts  (["node.Name"] = "dynakube.tenantUID")
// - not-mapped: there is NO infra-monitoring agent on the node => user processes will show up as individual 'fake' hosts (["node.Name"] = "-")
// - unknown: there SHOULD be a infra-monitoring agent on the node, but dynakube has NO tenantUID set => user processes will restart until this is fixed (node.Name not present in the map)
//
// Checks all the dynakubes with infra-monitoring against all the nodes (using the nodeSelector), creating the above mentioned mapping.
func (g *InitGenerator) getInfraMonitoringNodes(dk *dynatracev1beta1.DynaKube) (map[string]string, error) {
	var dks dynatracev1beta1.DynaKubeList
	if err := g.client.List(context.TODO(), &dks, client.InNamespace(g.namespace)); err != nil {
		return nil, errors.WithMessage(err, "failed to query DynaKubeList")
	}

	nodeInf, err := g.initIMNodes()
	if err != nil {
		return nil, err
	}

	for i := range dks.Items {
		status := &dks.Items[i].Status
		if dk != nil && dk.Name == dks.Items[i].Name {
			status = &dk.Status
		}
		if dks.Items[i].NeedsOneAgent() {
			tenantUUID := ""
			if status.ConnectionInfo.TenantUUID != "" {
				tenantUUID = status.ConnectionInfo.TenantUUID
			}
			nodeSelector := labels.SelectorFromSet(dks.Items[i].NodeSelector())
			for _, node := range nodeInf.nodes {
				nodeLabels := labels.Set(node.Labels)
				if nodeSelector.Matches(nodeLabels) {
					if tenantUUID != "" {
						nodeInf.imNodes[node.Name] = tenantUUID
					} else if !dk.FeatureIgnoreUnknownState() {
						delete(nodeInf.imNodes, node.Name)
					}
				}
			}
		}
	}

	return nodeInf.imNodes, nil
}

func (g *InitGenerator) initIMNodes() (nodeInfo, error) {
	var nodeList corev1.NodeList
	if err := g.client.List(context.TODO(), &nodeList); err != nil {
		return nodeInfo{}, err
	}
	imNodes := map[string]string{}
	for _, node := range nodeList.Items {
		imNodes[node.Name] = notMappedIM
	}
	return nodeInfo{nodeList.Items, imNodes}, nil
}

func (s *script) generate() (map[string][]byte, error) {
	var buf bytes.Buffer

	if err := scriptTmpl.Execute(&buf, s); err != nil {
		return nil, err
	}

	data := map[string][]byte{
		initScriptSecretField: buf.Bytes(),
	}

	if s.TrustedCAs != nil {
		data[trustedCAInitSecretField] = s.TrustedCAs
	}

	if s.Proxy != "" {
		data[proxyInitSecretField] = []byte(s.Proxy)
	}

	return data, nil
}
