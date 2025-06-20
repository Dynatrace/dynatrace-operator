package initgeneration

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	imNodes map[string]string
	nodes   []corev1.Node
}

func NewInitGenerator(client client.Client, apiReader client.Reader, namespace string) *InitGenerator {
	return &InitGenerator{
		client:    client,
		apiReader: apiReader,
		namespace: namespace,
	}
}

// GenerateForNamespace creates the init secret for namespace while only having the name of the corresponding dynakube
// Used by the podInjection webhook in case the namespace lacks the init secret.
func (g *InitGenerator) GenerateForNamespace(ctx context.Context, dk dynakube.DynaKube, targetNs string) error {
	log.Info("reconciling namespace init secret for", "namespace", targetNs)

	g.canWatchNodes = false

	data, err := g.generate(ctx, &dk)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(consts.AgentInitSecretName, targetNs, data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		return err
	}

	_, err = k8ssecret.Query(g.client, g.apiReader, log).CreateOrUpdate(ctx, secret)

	return errors.WithStack(err)
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *InitGenerator) GenerateForDynakube(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling namespace init secret for", "dynakube", dk.Name)

	g.canWatchNodes = true

	data, err := g.generate(ctx, dk)
	if err != nil {
		return errors.WithStack(err)
	}

	nsList, err := mapper.GetNamespacesForDynakube(ctx, g.apiReader, dk.Name)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)

	secret, err := k8ssecret.BuildForNamespace(consts.AgentInitSecretName, "", data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
	if err != nil {
		return err
	}

	err = k8ssecret.Query(g.client, g.apiReader, log).CreateOrUpdateForNamespaces(ctx, secret, nsList)
	if err != nil {
		return err
	}

	log.Info("done updating init secrets")

	return nil
}

func (g *InitGenerator) Cleanup(ctx context.Context, namespaces []corev1.Namespace) error {
	nsList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		nsList = append(nsList, ns.Name)
	}

	return k8ssecret.Query(g.client, g.apiReader, log).DeleteForNamespaces(ctx, consts.AgentInitSecretName, nsList)
}

// generate gets the necessary info the create the init secret data
func (g *InitGenerator) generate(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
	hostMonitoringNodes, err := g.getHostMonitoringNodes(dk)
	if err != nil {
		return nil, err
	}

	secretConfig, err := g.createSecretConfigForDynaKube(ctx, dk, hostMonitoringNodes)
	if err != nil {
		return nil, err
	}

	agCerts, err := dk.ActiveGateTLSCert(ctx, g.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	trustedCAs, err := dk.TrustedCAs(ctx, g.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	data, err := g.createSecretData(secretConfig, agCerts, trustedCAs)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (g *InitGenerator) createSecretConfigForDynaKube(ctx context.Context, dk *dynakube.DynaKube, hostMonitoringNodes map[string]string) (*startup.SecretConfig, error) {
	var tokens corev1.Secret
	if err := g.client.Get(ctx, client.ObjectKey{Name: dk.Tokens(), Namespace: g.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	var proxy string

	var err error
	if dk.NeedsOneAgentProxy() {
		proxy, err = dk.Proxy(ctx, g.apiReader)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	oneAgentNoProxyValues := []string{}

	if dk.NeedsCustomNoProxy() {
		oneAgentNoProxyValues = append(oneAgentNoProxyValues, dk.FF().GetNoProxy())
	}

	if dk.ActiveGate().IsRoutingEnabled() {
		oneAgentDNSEntry := capability.BuildHostEntries(*dk)
		oneAgentNoProxyValues = append(oneAgentNoProxyValues, oneAgentDNSEntry)
	}

	return &startup.SecretConfig{
		APIURL:              dk.Spec.APIURL,
		APIToken:            getAPIToken(tokens),
		PaasToken:           getPaasToken(tokens),
		TenantUUID:          dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID,
		Proxy:               proxy,
		NoProxy:             dk.FF().GetNoProxy(),
		OneAgentNoProxy:     strings.Join(oneAgentNoProxyValues, ","),
		NetworkZone:         dk.Spec.NetworkZone,
		SkipCertCheck:       dk.Spec.SkipCertCheck,
		HasHost:             dk.OneAgent().IsCloudNativeFullstackMode(),
		MonitoringNodes:     hostMonitoringNodes,
		HostGroup:           dk.OneAgent().GetHostGroup(),
		InitialConnectRetry: dk.FF().GetAgentInitialConnectRetry(dk.Spec.EnableIstio),
		EnforcementMode:     dk.FF().IsEnforcementMode(),
		ReadOnlyCSIDriver:   dk.FF().IsCSIVolumeReadOnly(),
		CSIMode:             dk.OneAgent().IsCSIAvailable(),
	}, nil
}

func getPaasToken(tokens corev1.Secret) string {
	if len(tokens.Data[dtclient.PaasToken]) != 0 {
		return string(tokens.Data[dtclient.PaasToken])
	}

	return string(tokens.Data[dtclient.APIToken])
}

func getAPIToken(tokens corev1.Secret) string {
	return string(tokens.Data[dtclient.APIToken])
}

// getHostMonitoringNodes creates a mapping between all the nodes and the tenantUID for the host-monitoring dynakube on that node.
// Possible mappings:
// - mapped: there is a host-monitoring agent on the node, and the dynakube has the tenantUID set => user processes will be grouped to the hosts  (["node.Name"] = "dynakube.tenantUID")
// - not-mapped: there is NO host-monitoring agent on the node => user processes will show up as individual 'fake' hosts (["node.Name"] = "-")
// - unknown: there SHOULD be a host-monitoring agent on the node, but dynakube has NO tenantUID set => user processes will restart until this is fixed (node.Name not present in the map)
//
// Checks all the dynakubes with host-monitoring against all the nodes (using the nodeSelector), creating the above mentioned mapping.
func (g *InitGenerator) getHostMonitoringNodes(dk *dynakube.DynaKube) (map[string]string, error) {
	tenantUUID := dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID

	imNodes := map[string]string{}
	if !dk.OneAgent().IsCloudNativeFullstackMode() {
		return imNodes, nil
	}

	if g.canWatchNodes {
		var err error

		imNodes, err = g.calculateImNodes(dk, tenantUUID)
		if err != nil {
			return nil, err
		}
	} else {
		updateImNodes(dk, tenantUUID, imNodes)
	}

	return imNodes, nil
}

func (g *InitGenerator) calculateImNodes(dk *dynakube.DynaKube, tenantUUID string) (map[string]string, error) {
	nodeInf, err := g.initIMNodes()
	if err != nil {
		return nil, err
	}

	nodeSelector := labels.SelectorFromSet(dk.OneAgent().GetNodeSelector(nil))
	updateNodeInfImNodes(dk, nodeInf, nodeSelector, tenantUUID)

	return nodeInf.imNodes, nil
}

func updateImNodes(dk *dynakube.DynaKube, tenantUUID string, imNodes map[string]string) {
	for nodeName := range dk.Status.OneAgent.Instances {
		if tenantUUID != "" {
			imNodes[nodeName] = tenantUUID
		} else if !dk.FF().IgnoreUnknownState() {
			delete(imNodes, nodeName)
		}
	}
}

func updateNodeInfImNodes(dk *dynakube.DynaKube, nodeInf nodeInfo, nodeSelector labels.Selector, tenantUUID string) {
	for _, node := range nodeInf.nodes {
		nodeLabels := labels.Set(node.Labels)
		if nodeSelector.Matches(nodeLabels) {
			if tenantUUID != "" {
				nodeInf.imNodes[node.Name] = tenantUUID
			} else if !dk.FF().IgnoreUnknownState() {
				delete(nodeInf.imNodes, node.Name)
			}
		}
	}
}

func (g *InitGenerator) initIMNodes() (nodeInfo, error) {
	var nodeList corev1.NodeList
	if err := g.client.List(context.TODO(), &nodeList); err != nil {
		return nodeInfo{}, err
	}

	imNodes := map[string]string{}
	for _, node := range nodeList.Items {
		imNodes[node.Name] = consts.AgentNoHostTenant
	}

	return nodeInfo{nodes: nodeList.Items, imNodes: imNodes}, nil
}

func (g *InitGenerator) createSecretData(secretConfig *startup.SecretConfig, agCerts []byte, cas []byte) (map[string][]byte, error) {
	jsonContent, err := json.Marshal(*secretConfig)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		consts.AgentInitSecretConfigField:   jsonContent,
		dynakube.ProxyKey:                   []byte(secretConfig.Proxy), // needed so that it can be mounted to the user's pod without directly reading the secret
		consts.ActiveGateCAsInitSecretField: agCerts,
		consts.TrustedCAsInitSecretField:    cas,
	}, nil
}
