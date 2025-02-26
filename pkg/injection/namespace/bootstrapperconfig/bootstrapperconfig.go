package bootstrapperconfig

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/enrichment/endpoint"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/ca"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/curl"
	"github.com/Dynatrace/dynatrace-bootstrapper/pkg/configure/oneagent/pmc"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/connectioninfo"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/processmoduleconfigsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BootstrapperInitGenerator manages the bootstrapper init secret generation for the user namespaces.
type BootstrapperInitGenerator struct {
	client        client.Client
	dtClient      dtclient.Client
	apiReader     client.Reader
	namespace     string
	secretName    string
	canWatchNodes bool
}

type nodeInfo struct {
	imNodes map[string]string
	nodes   []corev1.Node
}

func NewBootstrapperInitGenerator(client client.Client, apiReader client.Reader, dtClient dtclient.Client, namespace string) *BootstrapperInitGenerator {
	return &BootstrapperInitGenerator{
		client:     client,
		dtClient:   dtClient,
		apiReader:  apiReader,
		namespace:  namespace,
		secretName: consts.BootsTrapperInitSecretName,
	}
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *BootstrapperInitGenerator) GenerateForDynakube(ctx context.Context, dk *dynakube.DynaKube) error {
	log.Info("reconciling namespace bootstrapper init secret for", "dynakube", dk.Name)

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

	secret, err := k8ssecret.BuildForNamespace(consts.BootsTrapperInitSecretName, "", data, k8ssecret.SetLabels(coreLabels.BuildLabels()))
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

func (g *BootstrapperInitGenerator) Cleanup(ctx context.Context, namespaces []corev1.Namespace) error {
	nsList := make([]string, 0, len(namespaces))
	for _, ns := range namespaces {
		nsList = append(nsList, ns.Name)
	}

	return k8ssecret.Query(g.client, g.apiReader, log).DeleteForNamespaces(ctx, consts.BootsTrapperInitSecretName, nsList)
}

// generate gets the necessary info the create the init secret data
func (g *BootstrapperInitGenerator) generate(ctx context.Context, dk *dynakube.DynaKube) (map[string][]byte, error) {
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

	pmcSecret, err := g.preparePMCSecretForBootstrapper(ctx, *dk)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return map[string][]byte{
		pmc.InputFileName:        pmcSecret,
		ca.TrustedCertsInputFile: trustedCAs,
		ca.AgCertsInputFile:      agCerts,
		curl.InputFileName:       []byte(strconv.Itoa(secretConfig.InitialConnectRetry)),
		endpoint.InputFileName:   []byte(secretConfig.ApiToken),
	}, nil
}

func (g *BootstrapperInitGenerator) preparePMCSecretForBootstrapper(ctx context.Context, dk dynakube.DynaKube) ([]byte, error) {
	pmc, err := g.dtClient.GetProcessModuleConfig(ctx, 0)
	if err != nil {
		conditions.SetDynatraceApiError(dk.Conditions(), processmoduleconfigsecret.PmcConditionType, err)

		return nil, err
	}

	tenantToken, err := k8ssecret.GetDataFromSecretName(ctx, g.apiReader, types.NamespacedName{
		Name:      dk.OneAgent().GetTenantSecret(),
		Namespace: dk.Namespace,
	}, connectioninfo.TenantTokenKey, log)
	if err != nil {
		conditions.SetKubeApiError(dk.Conditions(), processmoduleconfigsecret.PmcConditionType, err)

		return nil, err
	}

	pmc = pmc.
		AddHostGroup(dk.OneAgent().GetHostGroup()).
		AddConnectionInfo(dk.Status.OneAgent.ConnectionInfoStatus, tenantToken).
		// set proxy explicitly empty, so old proxy settings get deleted where necessary
		AddProxy("")

	if dk.NeedsOneAgentProxy() {
		proxy, err := dk.Proxy(ctx, g.apiReader)
		if err != nil {
			conditions.SetKubeApiError(dk.Conditions(), processmoduleconfigsecret.PmcConditionType, err)

			return nil, err
		}

		pmc.AddProxy(proxy)

		multiCap := capability.NewMultiCapability(&dk)
		dnsEntry := capability.BuildDNSEntryPointWithoutEnvVars(dk.Name, dk.Namespace, multiCap)

		if dk.FeatureNoProxy() != "" {
			dnsEntry += "," + dk.FeatureNoProxy()
		}

		pmc.AddNoProxy(dnsEntry)
	}

	marshaled, err := json.Marshal(pmc)
	if err != nil {
		log.Info("could not marshal process module config")

		return nil, err
	}

	return marshaled, err
}

func (g *BootstrapperInitGenerator) createSecretConfigForDynaKube(ctx context.Context, dk *dynakube.DynaKube, hostMonitoringNodes map[string]string) (*startup.SecretConfig, error) {
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
		oneAgentNoProxyValues = append(oneAgentNoProxyValues, dk.FeatureNoProxy())
	}

	if dk.ActiveGate().IsRoutingEnabled() {
		multiCap := capability.NewMultiCapability(dk)
		oneAgentDNSEntry := capability.BuildDNSEntryPointWithoutEnvVars(dk.Name, dk.Namespace, multiCap)
		oneAgentNoProxyValues = append(oneAgentNoProxyValues, oneAgentDNSEntry)
	}

	return &startup.SecretConfig{
		ApiUrl:              dk.Spec.APIURL,
		ApiToken:            getAPIToken(tokens),
		PaasToken:           getPaasToken(tokens),
		TenantUUID:          dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID,
		Proxy:               proxy,
		NoProxy:             dk.FeatureNoProxy(),
		OneAgentNoProxy:     strings.Join(oneAgentNoProxyValues, ","),
		NetworkZone:         dk.Spec.NetworkZone,
		SkipCertCheck:       dk.Spec.SkipCertCheck,
		HasHost:             dk.OneAgent().IsCloudNativeFullstackMode(),
		MonitoringNodes:     hostMonitoringNodes,
		HostGroup:           dk.OneAgent().GetHostGroup(),
		InitialConnectRetry: dk.FeatureAgentInitialConnectRetry(),
		EnforcementMode:     dk.FeatureEnforcementMode(),
		ReadOnlyCSIDriver:   dk.FeatureReadOnlyCsiVolume(),
		CSIMode:             dk.OneAgent().IsCSIAvailable(),
	}, nil
}

func getPaasToken(tokens corev1.Secret) string {
	if len(tokens.Data[dtclient.PaasToken]) != 0 {
		return string(tokens.Data[dtclient.PaasToken])
	}

	return string(tokens.Data[dtclient.ApiToken])
}

func getAPIToken(tokens corev1.Secret) string {
	return string(tokens.Data[dtclient.ApiToken])
}

// getHostMonitoringNodes creates a mapping between all the nodes and the tenantUID for the host-monitoring dynakube on that node.
// Possible mappings:
// - mapped: there is a host-monitoring agent on the node, and the dynakube has the tenantUID set => user processes will be grouped to the hosts  (["node.Name"] = "dynakube.tenantUID")
// - not-mapped: there is NO host-monitoring agent on the node => user processes will show up as individual 'fake' hosts (["node.Name"] = "-")
// - unknown: there SHOULD be a host-monitoring agent on the node, but dynakube has NO tenantUID set => user processes will restart until this is fixed (node.Name not present in the map)
//
// Checks all the dynakubes with host-monitoring against all the nodes (using the nodeSelector), creating the above mentioned mapping.
func (g *BootstrapperInitGenerator) getHostMonitoringNodes(dk *dynakube.DynaKube) (map[string]string, error) {
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

func (g *BootstrapperInitGenerator) calculateImNodes(dk *dynakube.DynaKube, tenantUUID string) (map[string]string, error) {
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
		} else if !dk.FeatureIgnoreUnknownState() {
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
			} else if !dk.FeatureIgnoreUnknownState() {
				delete(nodeInf.imNodes, node.Name)
			}
		}
	}
}

func (g *BootstrapperInitGenerator) initIMNodes() (nodeInfo, error) {
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
