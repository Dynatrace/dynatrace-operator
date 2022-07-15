package initgeneration

import (
	"context"
	"encoding/json"
	"strconv"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	"github.com/Dynatrace/dynatrace-operator/src/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	dynakubeQuery kubeobjects.DynakubeQuery
}

type nodeInfo struct {
	nodes   []corev1.Node
	imNodes map[string]string
}

func NewInitGenerator(client client.Client, apiReader client.Reader, namespace string) *InitGenerator {
	return &InitGenerator{
		client:        client,
		apiReader:     apiReader,
		namespace:     namespace,
		dynakubeQuery: kubeobjects.NewDynakubeQuery(client, namespace),
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

	coreLabels := kubeobjects.NewCoreLabels(dk.Name, kubeobjects.WebhookComponentLabel)
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhook.SecretConfigName,
			Namespace: targetNs,
			Labels:    coreLabels.BuildMatchLabels(),
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}
	secretQuery := kubeobjects.NewSecretQuery(ctx, g.client, g.apiReader, log)

	return errors.WithStack(secretQuery.CreateOrUpdate(*secret))
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

	coreLabels := kubeobjects.NewCoreLabels(dk.Name, kubeobjects.WebhookComponentLabel)
	secretQuery := kubeobjects.NewSecretQuery(ctx, g.client, g.apiReader, log)

	for _, targetNs := range nsList {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name:      webhook.SecretConfigName,
				Namespace: targetNs.Name,
				Labels:    coreLabels.BuildMatchLabels(),
			},
			Data: data,
			Type: corev1.SecretTypeOpaque,
		}

		err = secretQuery.CreateOrUpdate(*secret)

		if err != nil {
			return errors.WithStack(err)
		}
	}

	log.Info("done updating init secrets")
	return nil
}

// generate gets the necessary info the create the init secret data
func (g *InitGenerator) generate(ctx context.Context, dk *dynatracev1beta1.DynaKube) (map[string][]byte, error) {
	kubeSystemUID, err := kubesystem.GetUID(g.apiReader)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	hostMonitoringNodes, err := g.getHostMonitoringNodes(dk)
	if err != nil {
		return nil, err
	}

	secretConfig, err := g.createSecretConfigForDynaKube(ctx, dk, kubeSystemUID, hostMonitoringNodes)
	if err != nil {
		return nil, err
	}

	data, err := g.createSecretData(secretConfig)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (g *InitGenerator) createSecretConfigForDynaKube(ctx context.Context, dynakube *dynatracev1beta1.DynaKube, kubeSystemUID types.UID, hostMonitoringNodes map[string]string) (*standalone.SecretConfig, error) {
	var tokens corev1.Secret
	if err := g.client.Get(ctx, client.ObjectKey{Name: dynakube.Tokens(), Namespace: g.namespace}, &tokens); err != nil {
		return nil, errors.WithMessage(err, "failed to query tokens")
	}

	dynakubeQuery := g.dynakubeQuery.WithContext(ctx)
	proxy, err := dynakubeQuery.Proxy(*dynakube)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	trustedCAs, err := dynakubeQuery.TrustedCAs(*dynakube)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	tlsCert, err := dynakubeQuery.TlsCert(*dynakube)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &standalone.SecretConfig{
		ApiUrl:              dynakube.Spec.APIURL,
		ApiToken:            getAPIToken(tokens),
		PaasToken:           getPaasToken(tokens),
		Proxy:               proxy,
		NetworkZone:         dynakube.Spec.NetworkZone,
		TrustedCAs:          string(trustedCAs),
		SkipCertCheck:       dynakube.Spec.SkipCertCheck,
		TenantUUID:          dynakube.Status.ConnectionInfo.TenantUUID,
		HasHost:             dynakube.CloudNativeFullstackMode(),
		MonitoringNodes:     hostMonitoringNodes,
		TlsCert:             tlsCert,
		HostGroup:           dynakube.HostGroup(),
		ClusterID:           string(kubeSystemUID),
		InitialConnectRetry: dynakube.FeatureAgentInitialConnectRetry(),
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
	hasHost := "false"
	if config.HasHost {
		hasHost = "true"
	}

	skipCertCheck := "false"
	if config.SkipCertCheck {
		skipCertCheck = "true"
	}

	toReturn := map[string][]byte{
		standalone.SecretConfigFieldName: jsonContent,
		"apiUrl":                         []byte(config.ApiUrl),
		"apiToken":                       []byte(config.ApiToken),
		"paasToken":                      []byte(config.PaasToken),
		dynatracev1beta1.ProxyKey:        []byte(config.Proxy), // needed so that it can be mounted to the user's pod without directly reading the secret
		"networkZone":                    []byte(config.NetworkZone),
		"trustedCAs":                     []byte(config.TrustedCAs),
		"skipCertCheck":                  []byte(skipCertCheck),
		"tenantUUID":                     []byte(config.TenantUUID),
		"hasHost":                        []byte(hasHost),
		"tlsCert":                        []byte(config.TlsCert),
		"hostGroup":                      []byte(config.HostGroup),
		"initialConnectRetry":            []byte(strconv.Itoa(config.InitialConnectRetry)),
		"clusterID":                      []byte(config.ClusterID),
	}

	config.MonitoringNodes["node1"] = "node1"
	config.MonitoringNodes["node2"] = "node2"
	config.MonitoringNodes["node3"] = "node3"

	for key, val := range config.MonitoringNodes {
		toReturn[key] = []byte(val)
	}

	return toReturn, nil
}
