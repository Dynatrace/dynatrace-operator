package script

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"math/rand"
	"net/url"
	"path/filepath"
	"strings"
	"text/template"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/logger"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	binDir    = "/mnt/bin"
	configDir = "/mnt/config"
	shareDir  = "/mnt/share"
)

//go:embed init.sh.tmpl
var scriptContent string

var scriptTmpl = template.Must(template.New("initScript").Parse(scriptContent))

var log = logger.NewDTLogger()

type scriptParams struct {
	InstallerMode  string
	InstallPath    string
	ShareDir       string
	ContainerCount int
	PodUID         string
	PodName        string
	PodBaseName    string
	Namespace      string
	HostTenant     string
	TenantUUID     string
	NodeName       string
	ClusterID      string
}

type InitGenerator struct {
	c   client.Client
	dk  *dynatracev1alpha1.DynaKube
	ns  *corev1.Namespace
	pod *corev1.Pod
}

func NewInitGenerator(client client.Client, dk *dynatracev1alpha1.DynaKube, ns *corev1.Namespace, pod *corev1.Pod) InitGenerator {
	return InitGenerator{
		c:   client,
		dk:  dk,
		ns:  ns,
		pod: pod,
	}
}

func (ig InitGenerator) NewScript(ctx context.Context) error {
	tokenName := ig.dk.Tokens()
	if !ig.dk.Spec.CodeModules.Enabled {
		_ = ensureSecretDeleted(ig.c, tokenName, ig.ns.Name)
		return nil // TODO: return someting
	}

	trustedCAs, err := ig.trustedCAs(ctx)
	if err != nil {
		return err
	}

	proxy, err := ig.proxy(context.TODO())
	if err != nil {
		return err
	}

	installerMode, err := ig.installerMode(trustedCAs, proxy)
	if err != nil {
		return err
	}
	hostTenant, err := ig.hostTenant(ctx)
	if err != nil {
		return err
	}
	clusterID, err := ig.clusterId(ctx)
	if err != nil {
		return err
	}
	params := scriptParams{
		InstallerMode:  installerMode,
		InstallPath:    ig.installPath(),
		ShareDir:       shareDir,
		ContainerCount: ig.containerCount(),
		PodUID:         ig.podUID(),
		PodName:        ig.podName(),
		PodBaseName:    ig.basePodName(),
		Namespace:      ig.ns.Name,
		TenantUUID:     ig.dk.ConnectionInfo().TenantUUID,
		HostTenant:     hostTenant,
		NodeName:       ig.nodeName(),
		ClusterID:      clusterID,
	}

	script, err := ig.generate(trustedCAs, proxy, params)
	if err != nil {
		return err
	}
	err = utils.CreateOrUpdateSecretIfNotExists(ig.c, dtwebhook.SecretConfigName, ig.ns.Name, script, corev1.SecretTypeOpaque, log)
	if err != nil {
		return err
	}

	return nil
}

func (ig InitGenerator) installerMode(trustedCA []byte, proxy string) (string, error) {
	if ig.mode() != "installer" {
		return "", nil
	}
	archivePath := filepath.Join(binDir, fmt.Sprintf("tmp.%d", rand.Intn(1000)))
	curlParams := []string{
		"--silent",
		fmt.Sprintf("--output \"%s\"", archivePath),
	}

	if url := ig.installerURL(); url != "" {
		curlParams = append(curlParams, url)
	} else {
		url := fmt.Sprintf("%s/v1/deployment/installer/agent/unix/paas/latest?flavor=%s&include=%s&bitness=64",
			ig.apiURL(),
			ig.flavor(),
			ig.technologies())
		paasToken, err := ig.paasToken(context.TODO())
		if err != nil {
			return "", err
		}
		header := fmt.Sprintf("--header Authorization: Api-Token %s", paasToken)
		curlParams = append(curlParams, url, header)
	}

	if ig.dk.Spec.SkipCertCheck {
		curlParams = append(curlParams, "--insecure")
	}

	if trustedCA != nil {
		curlParams = append(curlParams, "--cacert %s/ca.pem", configDir)
	}

	if proxy != "" {
		curlParams = append(curlParams, "--proxy %s", proxy)
	}

	curl := strings.Join(curlParams, " ")
	failCode := ig.failCode()

	curlCommand := fmt.Sprintf(`
	echo "Downloading OneAgent package..."
	if ! curl "%s"; then
		echo "Failed to download the OneAgent package."
		exit "%d"
	fi
	`, curl, failCode)

	unzipCommand := fmt.Sprintf(`
	echo "Unpacking OneAgent package..."
	if ! unzip -o -d "%s" "%s"; then
		echo "Failed to unpack the OneAgent package."
		mv "%s" "%s/package.zip"
		exit "%d"
	fi
	`, binDir, archivePath, archivePath, binDir, failCode)
	return (curlCommand + "\n" + unzipCommand), nil
}

func (ig InitGenerator) imnodes(ctx context.Context) (map[string]string, error) {
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

func (ig InitGenerator) paasToken(ctx context.Context) (string, error) {
	var tokens corev1.Secret
	if err := ig.c.Get(ctx, client.ObjectKey{Name: ig.dk.Tokens(), Namespace: ig.ns.Name}, &tokens); err != nil {
		return "", errors.WithMessage(err, "failed to query tokens")
	}
	return string(tokens.Data[utils.DynatracePaasToken]), nil
}

func (ig InitGenerator) clusterId(ctx context.Context) (string, error) {
	var kubeSystemNS corev1.Namespace
	if err := ig.c.Get(ctx, client.ObjectKey{Name: "kube-system"}, &kubeSystemNS); err != nil {
		return "", fmt.Errorf("failed to query for cluster ID: %w", err)
	}
	return string(kubeSystemNS.UID), nil
}

func (ig InitGenerator) proxy(ctx context.Context) (string, error) {
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

func (ig InitGenerator) trustedCAs(ctx context.Context) ([]byte, error) {
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

func (ig InitGenerator) hostTenant(ctx context.Context) (string, error) {
	imNodes, err := ig.imnodes(ctx)
	if err != nil {
		return "", err
	}
	return imNodes[ig.nodeName()], nil
}

func (ig InitGenerator) apiURL() string {
	return ig.dk.Spec.APIURL
}

func (ig InitGenerator) flavor() string {
	return dtclient.FlavorMultidistro
}

func (ig InitGenerator) containerCount() int {
	return len(ig.pod.Spec.Containers)
}

func (ig InitGenerator) technologies() string {
	return url.QueryEscape(utils.GetField(ig.pod.Annotations, dtwebhook.AnnotationTechnologies, "all"))
}

func (ig InitGenerator) installPath() string {
	return utils.GetField(ig.pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)
}

func (ig InitGenerator) installerURL() string {
	return utils.GetField(ig.pod.Annotations, dtwebhook.AnnotationInstallerUrl, "")
}

func (ig InitGenerator) failCode() int {
	failurePolicy := utils.GetField(ig.pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")
	if failurePolicy == "fail" {
		return 1
	}
	return 0
}

func (ig InitGenerator) mode() string {
	mode := "provisioned"
	if ig.dk.Spec.CodeModules.Volume.EmptyDir != nil {
		mode = "installer"
	}
	return mode
}

func (ig InitGenerator) basePodName() string {
	basePodName := ig.pod.GenerateName
	if basePodName == "" {
		basePodName = ig.pod.Name
	}
	// Only include up to the last dash character, exclusive.
	if p := strings.LastIndex(basePodName, "-"); p != -1 {
		basePodName = basePodName[:p]
	}
	return basePodName
}

func (ig InitGenerator) podName() string {
	return ig.pod.Name
}

func (ig InitGenerator) podUID() string {
	return string(ig.pod.UID)
}

func (ig InitGenerator) nodeName() string {
	return ig.pod.Spec.NodeName
}

func (ig InitGenerator) generate(trustedCA []byte, proxy string, params scriptParams) (map[string][]byte, error) {
	var buf bytes.Buffer

	if err := scriptTmpl.Execute(&buf, params); err != nil {
		return nil, err
	}

	data := map[string][]byte{
		"init.sh": buf.Bytes(),
	}

	if trustedCA != nil {
		data["ca.pem"] = trustedCA
	}

	if proxy != "" {
		data["proxy"] = []byte(proxy)
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
