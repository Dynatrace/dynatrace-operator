package script

import (
	b64 "encoding/base64"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	binDir    = "/mnt/bin"
	configDir = "/mnt/config"
	shareDir  = "/mnt/share"
)

//go:embed init.sh.tmpl
var scriptContent string

//go:embed containerConfBase.sh.tmpl
var containerConfBase string

//go:embed containerConfWithTenant.sh.tmpl
var containerConfWithTenant string

var scriptTmpl = template.Must(template.New("initScript").Parse(scriptContent))

var confBaseTmpl = template.Must(template.New("confBase").Parse(containerConfBase))

var confTenantTmpl = template.Must(template.New("confTenant").Parse(containerConfWithTenant))

var log = logger.NewDTLogger()

type scriptParams struct {
	InstallerMode   string
	LDSOPreload     string
	CreateConfFiles string
}

type confParams struct {
	ConfFile string
	ContainerName string
	ImageName string
	PodBaseName string
	Namespace string
	ClusterId string
	HostTenant string
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

func (ig InitGenerator) NewScript(ctx context.Context) (map[string]string, error) {
	trustedCAs, err := ig.trustedCAs(ctx)
	if err != nil {
		return nil, err
	}

	proxy, err := ig.proxy(context.TODO())
	if err != nil {
		return nil, err
	}

	installerMode, err := ig.installerMode(trustedCAs, proxy)
	if err != nil {
		return nil, err
	}
	hostTenant, err := ig.hostTenant(ctx)
	if err != nil {
		return nil, err
	}
	clusterID, err := ig.clusterId(ctx)
	if err != nil {
		return nil, err
	}
	params := scriptParams{
		InstallerMode:   installerMode,
		LDSOPreload: ig.ldSOPreload(),
		CreateConfFiles: ig.createContainerConfFile(hostTenant, clusterID),
	}

	script, err := ig.generate(trustedCAs, proxy, params)
	if err != nil {
		return nil, err
	}

	return script, nil
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

func (ig InitGenerator) ldSOPreload() string {
	return fmt.Sprintf("echo -n \"%s/agent/lib64/liboneagentproc.so\" >> \"%s/ld.so.preload\"", ig.installPath(), shareDir)
}

func (ig InitGenerator) createContainerConfFile(hostTenant, clusterId string) string {
	var createConfFileCmd bytes.Buffer
	var err error
	for i := range ig.pod.Spec.Containers {
		container := &ig.pod.Spec.Containers[i]
		params := confParams{
			ConfFile: getConfFilePath(container.Name),
			ContainerName: container.Name,
			ImageName: container.Image,
	        PodBaseName: ig.basePodName(),
	        Namespace: ig.ns.Name,
		}
		if hostTenant == ig.dk.ConnectionInfo().TenantUUID {
	        params.ClusterId = clusterId
			params.HostTenant = hostTenant
			err = confTenantTmpl.Execute(&createConfFileCmd, params)
		} else {
			err = confBaseTmpl.Execute(&createConfFileCmd, params)
		}
	}
	if err != nil {
		log.Error(err, "Something broke")
	}
	return createConfFileCmd.String()
}

func getConfFilePath(containerName string) string {
	return filepath.Join(shareDir, fmt.Sprintf("container_%s.conf", containerName))
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

func (ig InitGenerator) nodeName() string {
	return ig.pod.Spec.NodeName
}

func (ig InitGenerator) generate(trustedCA []byte, proxy string, params scriptParams) (map[string]string, error) {
	var buf bytes.Buffer

	if err := scriptTmpl.Execute(&buf, params); err != nil {
		return nil, err
	}

	data := map[string]string{
		"init.sh": b64.StdEncoding.EncodeToString(buf.Bytes()),
	}

	if trustedCA != nil {
		data["ca.pem"] = string(trustedCA)
	}

	if proxy != "" {
		data["proxy"] = proxy
	}

	return data, nil
}
