//go:build integration
// +build integration

package integrationtests

// This file includes utilities to start an environment with API Server, and a configured reconciler.

import (
	"context"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/Dynatrace/dynatrace-operator/src/version"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	DefaultTestAPIURL    = "https://ENVIRONMENTID.live.dynatrace.com/api"
	DefaultTestNamespace = "dynatrace"
)

func init() {
	os.Setenv("POD_NAMESPACE", DefaultTestNamespace)
}

type ControllerTestEnvironment struct {
	CommunicationHosts []string
	Client             client.Client
	Reconciler         *dynakube.DynakubeController

	server *envtest.Environment
}

func TestFindIstio(t *testing.T) {
	fs := afero.NewMemMapFs()
	istioPath := filepath.Join(build.Default.GOPATH, "pkg", "mod", "istio.io")
	err := fs.MkdirAll(istioPath, 0755)

	require.NoError(t, err)

	err = fs.MkdirAll(filepath.Join(istioPath, "api@v0.0.0-20220721211444-f06fcca0ad6c", "kubernetes"), 0755)
	require.NoError(t, err)

	err = fs.MkdirAll(filepath.Join(istioPath, "api@v0.0.0-20220110211529-694b7b802a22", "kubernetes"), 0755)
	require.NoError(t, err)

	err = fs.MkdirAll(filepath.Join(istioPath, "api@v0.0.0-20201125194658-3cee6a1d3ab4", "kubernetes"), 0755)
	require.NoError(t, err)

	err = fs.MkdirAll(filepath.Join(istioPath, "client-go@v1.12.1", "kubernetes"), 0755)
	require.NoError(t, err)

	latestDirectory, err := findLatestIstioDirectory(fs, istioPath)

	assert.NoError(t, err)
	assert.Equal(t, "api@v0.0.0-20220721211444-f06fcca0ad6c", latestDirectory)
}

func findLatestIstioDirectory(filesystem afero.Fs, istioPath string) (string, error) {
	directories, err := afero.ReadDir(filesystem, istioPath)
	if err != nil {
		return "", errors.WithStack(err)
	}

	var latestDirectory string
	var latestVersion version.SemanticVersion
	var directoryVersion version.SemanticVersion

	for _, directory := range directories {
		if !directory.IsDir() || !strings.Contains(directory.Name(), "api") {
			continue
		}

		// The 'version' package is made to work with the versioning of the OneAgent
		// which is slightly different from the one used by the istio/api packages.
		// The lines below adjust the istio/api version, so it's compatible with our existing
		// semver implementation. For a better understanding, checkout the regex in the 'version' package.
		trimmedVersion := strings.ReplaceAll(directory.Name(), "api@v", "")
		trimmedVersion = trimmedVersion[:strings.LastIndex(trimmedVersion, "-")]
		trimmedVersion = strings.ReplaceAll(trimmedVersion, "-", ".")
		trimmedVersion = trimmedVersion + "-0"
		directoryVersion, err = version.ExtractSemanticVersion(trimmedVersion)

		if err != nil {
			continue
		}

		if latestDirectory == "" || version.CompareSemanticVersions(latestVersion, directoryVersion) < 0 {
			latestVersion = directoryVersion
			latestDirectory = directory.Name()
		}
	}

	return latestDirectory, nil
}

func newTestEnvironment() (*ControllerTestEnvironment, error) {
	istioPath := filepath.Join(build.Default.GOPATH, "pkg", "mod", "istio.io")
	latestIstioDirectory, err := findLatestIstioDirectory(afero.NewOsFs(), istioPath)

	if err != nil {
		return nil, err
	}

	environment := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "..", "config", "crd", "bases"),
			filepath.Join(istioPath, latestIstioDirectory, "kubernetes"),
		},
	}
	kubernetesAPIServer := environment.ControlPlane.GetAPIServer()

	arguments := kubernetesAPIServer.Configure()
	arguments.Set("--allow-privileged")

	cfg, err := environment.Start()
	if err != nil {
		return nil, err
	}

	kubernetesClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	if err = kubernetesClient.Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: DefaultTestNamespace,
		}}); err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	if err = kubernetesClient.Create(context.TODO(), buildDynatraceClientSecret()); err != nil {
		errStop := kubernetesAPIServer.Stop()
		if errStop != nil {
			return nil, fmt.Errorf("%s\n%s", err.Error(), errStop.Error())
		}
		return nil, err
	}

	communicationHosts := []string{
		"https://endpoint1.test.com/communication",
		"https://endpoint2.test.com/communication",
	}
	testEnvironment := &ControllerTestEnvironment{
		server:             environment,
		Client:             kubernetesClient,
		CommunicationHosts: communicationHosts,
	}
	testEnvironment.Reconciler = dynakube.NewDynaKubeController(
		kubernetesClient, kubernetesClient, scheme.Scheme,
		mockDynatraceClientFunc(&testEnvironment.CommunicationHosts), cfg)

	return testEnvironment, nil
}

func (e *ControllerTestEnvironment) Stop() error {
	return e.server.Stop()
}

func (e *ControllerTestEnvironment) AddOneAgent(n string, s *dynatracev1beta1.DynaKubeSpec) (*dynatracev1beta1.DynaKube, error) {
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n,
			Namespace: DefaultTestNamespace,
		},
		Spec: *s,
	}

	return instance, e.Client.Create(context.TODO(), instance)
}

func newReconciliationRequest(oaName string) reconcile.Request {
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      oaName,
			Namespace: DefaultTestNamespace,
		},
	}
}

func mockDynatraceClientFunc(communicationHosts *[]string) dynakube.DynatraceClientFunc {
	return func(dynakube.DynatraceClientProperties) (dtclient.Client, error) {
		commHosts := make([]dtclient.CommunicationHost, len(*communicationHosts))
		for i, c := range *communicationHosts {
			commHosts[i] = dtclient.CommunicationHost{Protocol: "https", Host: c, Port: 443}
		}

		connInfo := dtclient.ConnectionInfo{
			TenantUUID:         "asdf",
			CommunicationHosts: commHosts,
		}

		dtc := new(dtclient.MockDynatraceClient)
		dtc.On("GetLatestAgentVersion", "unix", "default").Return("17", nil)
		dtc.On("GetLatestAgentVersion", "unix", "paas").Return("18", nil)
		dtc.On("GetConnectionInfo").Return(connInfo, nil)
		dtc.On("GetCommunicationHostForClient").Return(dtclient.CommunicationHost{
			Protocol: "https",
			Host:     DefaultTestAPIURL,
			Port:     443,
		}, nil)
		dtc.On("GetTokenScopes", "42").Return(dtclient.TokenScopes{dtclient.TokenScopeInstallerDownload}, nil)
		dtc.On("GetTokenScopes", "43").Return(dtclient.TokenScopes{dtclient.TokenScopeDataExport}, nil)

		return dtc, nil
	}
}

func buildDynatraceClientSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "token-test",
			Namespace: DefaultTestNamespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"paasToken": []byte("42"),
			"apiToken":  []byte("43"),
		},
	}
}
