package kubesystem

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

const (
	float64BitSize = 64
)

const errorConfigIsNil = "rest config for Kubernetes version provider is nil"

type VersionProvider interface {
	Major() (string, error)
	Minor() (string, error)
}

type discoveryVersionProvider struct {
	config      *rest.Config
	versionInfo *version.Info
}

func NewVersionProvider(config *rest.Config) VersionProvider {
	return &discoveryVersionProvider{
		config: config,
	}
}

func (versionProvider *discoveryVersionProvider) Major() (string, error) {
	versionInfo, err := versionProvider.getVersionInfo()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return versionInfo.Major, nil
}

func (versionProvider *discoveryVersionProvider) Minor() (string, error) {
	versionInfo, err := versionProvider.getVersionInfo()
	if err != nil {
		return "", errors.WithStack(err)
	}
	return versionInfo.Minor, nil
}

func (versionProvider *discoveryVersionProvider) getVersionInfo() (*version.Info, error) {
	if versionProvider.versionInfo != nil {
		return versionProvider.versionInfo, nil
	}

	if versionProvider.config == nil {
		return nil, errors.New(errorConfigIsNil)
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(versionProvider.config)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	versionInfo, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	versionProvider.versionInfo = versionInfo
	return versionInfo, nil
}

func KubernetesVersionAsFloat(major, minor string) float64 {
	versionString := combinedKubernetesVersion(major, minor)
	return parseVersionString(versionString)
}

func parseVersionString(versionString string) float64 {
	parsedVersion, err := strconv.ParseFloat(versionString, float64BitSize)
	if err != nil {
		return 0
	}
	if parsedVersion < 0 {
		return 0
	}
	return parsedVersion
}

func combinedKubernetesVersion(major, minor string) string {
	combinedVersion := fmt.Sprintf("%s.%s", major, minor)
	neitherNumberNorDot := regexp.MustCompile("[^0-9.]")
	return neitherNumberNorDot.ReplaceAllString(combinedVersion, "")
}
