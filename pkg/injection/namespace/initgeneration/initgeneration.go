package initgeneration

import (
	"context"
	"encoding/json"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta1/dynakube"
	dtclient "github.com/Dynatrace/dynatrace-operator/pkg/clients/dynatrace"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/activegate/capability"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/namespace/mapper"
	"github.com/Dynatrace/dynatrace-operator/pkg/injection/startup"
	k8slabels "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	k8ssecret "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/secret"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
func (g *InitGenerator) GenerateForNamespace(ctx context.Context, dk dynatracev1beta1.DynaKube, targetNs string) error {
	log.Info("reconciling namespace init secret for", "namespace", targetNs)

	g.canWatchNodes = false

	data, err := g.generate(ctx, &dk)
	if err != nil {
		return errors.WithStack(err)
	}

	coreLabels := k8slabels.NewCoreLabels(dk.Name, k8slabels.WebhookComponentLabel)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.AgentInitSecretName,
			Namespace: targetNs,
			Labels:    coreLabels.BuildMatchLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
	secretQuery := k8ssecret.NewQuery(ctx, g.client, g.apiReader, log)

	err = secretQuery.CreateOrUpdate(*secret)

	return errors.WithStack(err)
}

// GenerateForDynakube creates/updates the init secret for EVERY namespace for the given dynakube.
// Used by the dynakube controller during reconcile.
func (g *InitGenerator) GenerateForDynakube(ctx context.Context, dk *dynatracev1beta1.DynaKube) error {
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
	secretQuery := k8ssecret.NewQuery(ctx, g.client, g.apiReader, log)
	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:   consts.AgentInitSecretName,
			Labels: coreLabels.BuildMatchLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}

	err = secretQuery.CreateOrUpdateForNamespaces(secret, nsList)
	if err != nil {
		return err
	}

	log.Info("done updating init secrets")

	return nil
}

// generate gets the necessary info the create the init secret data
func (g *InitGenerator) generate(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
	hostMonitoringNodes, err := g.getHostMonitoringNodes(dk)
	if err != nil {
		return nil, err
	}

	trustedCAs, err := dk.TrustedCAs(ctx, g.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	tlsCert, err := dk.ActiveGateTlsCert(ctx, g.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if tlsCert != nil {
		if trustedCAs == nil {
			trustedCAs = tlsCert
		} else {
			trustedCAs = append(trustedCAs, byte('\n'))
			trustedCAs = append(trustedCAs, tlsCert...)
		}
	}

	secretConfig, err := g.createSecretConfigForDynaKube(ctx, dk, hostMonitoringNodes)
	if err != nil {
		return nil, err
	}

	data, err := g.createSecretData(secretConfig, trustedCAs)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (g *InitGenerator) createSecretConfigForDynaKube(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, hostMonitoringNodes map[string]string) (*startup.SecretConfig, error) {
	var tokens corev1.Secret
	if err := g.client.Get(ctx, client.ObjectKey{Name: dynakube.Tokens(), Namespace: g.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	var proxy string

	var err error
	if dynakube.NeedsOneAgentProxy() {
		proxy, err = dynakube.Proxy(ctx, g.apiReader)
		if err != nil {
			return nil, errors.WithStack(err)
		}
	}

	oneAgentNoProxy := ""

	if dynakube.NeedsActiveGate() {
		multiCap := capability.NewMultiCapability(dynakube)
		oneAgentNoProxy = capability.BuildDNSEntryPointWithoutEnvVars(dynakube.Name, dynakube.Namespace, multiCap)
	}

	return &startup.SecretConfig{
		ApiUrl:              dynakube.Spec.APIURL,
		ApiToken:            getAPIToken(tokens),
		PaasToken:           getPaasToken(tokens),
		TenantUUID:          dynakube.Status.OneAgent.ConnectionInfoStatus.TenantUUID,
		Proxy:               proxy,
		NoProxy:             dynakube.FeatureNoProxy(),
		OneAgentNoProxy:     oneAgentNoProxy,
		NetworkZone:         dynakube.Spec.NetworkZone,
		SkipCertCheck:       dynakube.Spec.SkipCertCheck,
		HasHost:             dynakube.CloudNativeFullstackMode(),
		MonitoringNodes:     hostMonitoringNodes,
		HostGroup:           dynakube.HostGroup(),
		InitialConnectRetry: dynakube.FeatureAgentInitialConnectRetry(),
		EnforcementMode:     dynakube.FeatureEnforcementMode(),
		ReadOnlyCSIDriver:   dynakube.FeatureReadOnlyCsiVolume(),
		CSIMode:             dynakube.NeedsCSIDriver(),
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
func (g *InitGenerator) getHostMonitoringNodes(dk *dynatracev1beta1.DynaKube) (map[string]string, error) {
	tenantUUID := dk.Status.OneAgent.ConnectionInfoStatus.TenantUUID

	imNodes := map[string]string{}
	if !dk.CloudNativeFullstackMode() {
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

func (g *InitGenerator) calculateImNodes(dk *dynatracev1beta1.DynaKube, tenantUUID string) (map[string]string, error) {
	nodeInf, err := g.initIMNodes()
	if err != nil {
		return nil, err
	}

	nodeSelector := labels.SelectorFromSet(dk.NodeSelector())
	updateNodeInfImNodes(dk, nodeInf, nodeSelector, tenantUUID)

	return nodeInf.imNodes, nil
}

func updateImNodes(dk *dynatracev1beta1.DynaKube, tenantUUID string, imNodes map[string]string) {
	for nodeName := range dk.Status.OneAgent.Instances {
		if tenantUUID != "" {
			imNodes[nodeName] = tenantUUID
		} else if !dk.FeatureIgnoreUnknownState() {
			delete(imNodes, nodeName)
		}
	}
}

func updateNodeInfImNodes(dk *dynatracev1beta1.DynaKube, nodeInf nodeInfo, nodeSelector labels.Selector, tenantUUID string) {
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

func (g *InitGenerator) createSecretData(secretConfig *startup.SecretConfig, trustedCAs []byte) (map[string][]byte, error) {
	jsonContent, err := json.Marshal(*secretConfig)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		consts.AgentInitSecretConfigField:     jsonContent,
		consts.AgentInitSecretTrustedCAsField: trustedCAs,
		dynatracev1beta1.ProxyKey:             []byte(secretConfig.Proxy), // needed so that it can be mounted to the user's pod without directly reading the secret
	}, nil
}
