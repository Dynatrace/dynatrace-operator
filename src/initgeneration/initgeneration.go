package initgeneration

import (
	"context"
	"encoding/json"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// InitGenerator manages the init secret generation for the user namespaces.
type InitGenerator struct {
	client        client.Client
	apiReader     client.Reader
	namespace     string
	canWatchNodes bool
}

type nodeInfo struct {
	nodes   []corev1.Node
	imNodes map[string]string
}

func NewInitGenerator(client client.Client, apiReader client.Reader, ns string) *InitGenerator {
	return &InitGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: ns,
	}
}

// GenerateForNamespace creates the init secret for namespace while only having the name of the corresponding dynakube
// Used by the podInjection webhook in case the namespace lacks the init secret.
func (g *InitGenerator) GenerateForNamespace(ctx context.Context, dk dynatracev1beta1.DynaKube, targetNs string) (bool, error) {
	log.Info("reconciling namespace init secret for", "namespace", targetNs)
	g.canWatchNodes = false
	data, err := g.generate(ctx, &dk)
	if err != nil {
		return false, err
	}
	labels := kubeobjects.CommonLabels(dk.Name, kubeobjects.WebhookComponentLabel)
	return kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, webhook.SecretConfigName, targetNs, data, labels, corev1.SecretTypeOpaque, log)
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *InitGenerator) GenerateForDynakube(ctx context.Context, dk *dynatracev1beta1.DynaKube) (bool, error) {
	log.Info("reconciling namespace init secret for", "dynakube", dk.Name)
	g.canWatchNodes = true
	data, err := g.generate(ctx, dk)
	if err != nil {
		return false, err
	}

	anyUpdate := false
	nsList, err := mapper.GetNamespacesForDynakube(ctx, g.apiReader, dk.Name)
	if err != nil {
		return false, err
	}
	labels := kubeobjects.CommonLabels(dk.Name, kubeobjects.WebhookComponentLabel)
	for _, targetNs := range nsList {
		if upd, err := kubeobjects.CreateOrUpdateSecretIfNotExists(g.client, g.apiReader, webhook.SecretConfigName, targetNs.Name, data, labels, corev1.SecretTypeOpaque, log); err != nil {
			return false, err
		} else if upd {
			anyUpdate = true
		}
	}
	log.Info("done updating init secrets")
	return anyUpdate, nil
}

// generate gets the necessary info the create the init secret data
func (g *InitGenerator) generate(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
	kubeSystemUID, err := kubesystem.GetUID(g.apiReader)
	if err != nil {
		return nil, err
	}

	hostMonitoringNodes, err := g.getHostMonitoringNodes(dk)
	if err != nil {
		return nil, err
	}

	secretConfig, err := g.prepareSecretConfigForDynaKube(dk, kubeSystemUID, hostMonitoringNodes)
	if err != nil {
		return nil, err
	}

	data, err := g.createSecretData(secretConfig)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (g *InitGenerator) prepareSecretConfigForDynaKube(dk *dynatracev1beta1.DynaKube, kubeSystemUID types.UID, hostMonitoringNodes map[string]string) (*standalone.SecretConfig, error) {
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
			proxy = string(ps.Data[proxyKey])
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
		trustedCAs = []byte(cam.Data[trustedCAKey])
	}

	var tlsCert string
	if dk.HasActiveGateTLS() {
		var tlsSecret corev1.Secret
		if err := g.client.Get(context.TODO(), client.ObjectKey{Name: dk.Spec.ActiveGate.TlsSecretName, Namespace: g.namespace}, &tlsSecret); err != nil {
			return nil, fmt.Errorf("failed to query tls secret: %w", err)
		}
		tlsCert = string(tlsSecret.Data[tlsCertKey])
	}

	return &standalone.SecretConfig{
		ApiUrl:          dk.Spec.APIURL,
		ApiToken:        getAPIToken(tokens),
		PaasToken:       getPaasToken(tokens),
		Proxy:           proxy,
		NetworkZone:     dk.Spec.NetworkZone,
		TrustedCAs:      string(trustedCAs),
		SkipCertCheck:   dk.Spec.SkipCertCheck,
		TenantUUID:      dk.Status.ConnectionInfo.TenantUUID,
		HasHost:         dk.CloudNativeFullstackMode(),
		MonitoringNodes: hostMonitoringNodes,
		TlsCert:         tlsCert,
		HostGroup:       dk.HostGroup(),
		ClusterID:       string(kubeSystemUID),
	}, nil
}

func getPaasToken(tokens corev1.Secret) string {
	if len(tokens.Data[dtclient.DynatracePaasToken]) != 0 {
		return string(tokens.Data[dtclient.DynatracePaasToken])
	}
	return string(tokens.Data[dtclient.DynatraceApiToken])
}

func getAPIToken(tokens corev1.Secret) string {
	return string(tokens.Data[dtclient.DynatraceApiToken])
}

// getHostMonitoringNodes creates a mapping between all the nodes and the tenantUID for the host-monitoring dynakube on that node.
// Possible mappings:
// - mapped: there is a host-monitoring agent on the node, and the dynakube has the tenantUID set => user processes will be grouped to the hosts  (["node.Name"] = "dynakube.tenantUID")
// - not-mapped: there is NO host-monitoring agent on the node => user processes will show up as individual 'fake' hosts (["node.Name"] = "-")
// - unknown: there SHOULD be a host-monitoring agent on the node, but dynakube has NO tenantUID set => user processes will restart until this is fixed (node.Name not present in the map)
//
// Checks all the dynakubes with host-monitoring against all the nodes (using the nodeSelector), creating the above mentioned mapping.
func (g *InitGenerator) getHostMonitoringNodes(dk *dynatracev1beta1.DynaKube) (map[string]string, error) {

	imNodes := map[string]string{}
	if !dk.CloudNativeFullstackMode() {
		return imNodes, nil
	}
	tenantUUID := dk.Status.ConnectionInfo.TenantUUID
	if g.canWatchNodes {
		nodeInf, err := g.initIMNodes()
		if err != nil {
			return nil, err
		}
		nodeSelector := labels.SelectorFromSet(dk.NodeSelector())
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
		imNodes = nodeInf.imNodes
	} else {
		for nodeName := range dk.Status.OneAgent.Instances {
			if tenantUUID != "" {
				imNodes[nodeName] = tenantUUID
			} else if !dk.FeatureIgnoreUnknownState() {
				delete(imNodes, nodeName)
			}
		}
	}
	return imNodes, nil
}

func (g *InitGenerator) initIMNodes() (nodeInfo, error) {
	var nodeList corev1.NodeList
	if err := g.client.List(context.TODO(), &nodeList); err != nil {
		return nodeInfo{}, err
	}
	imNodes := map[string]string{}
	for _, node := range nodeList.Items {
		imNodes[node.Name] = standalone.NoHostTenant
	}
	return nodeInfo{nodeList.Items, imNodes}, nil
}

func (g *InitGenerator) createSecretData(config *standalone.SecretConfig) (map[string][]byte, error) {
	jsonContent, err := json.Marshal(*config)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{
		standalone.SecretConfigFieldName: jsonContent,
	}, nil
}
