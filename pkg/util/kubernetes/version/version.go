package version

import (
	"fmt"

	k8sversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
)

func GetServerVersion(discoveryClient discovery.ServerVersionInterface) (*k8sversion.Info, error) {
	return discoveryClient.ServerVersion()
}

func GetFormattedServerVersion(discoveryClient discovery.ServerVersionInterface) (string, error) {
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(
		"major: %s\nminor: %s\ngitVersion: %s\ngitCommit: %s\nbuildDate: %s\ngoVersion: %s\nplatform: %s\n",
		serverVersion.Major,
		serverVersion.Minor,
		serverVersion.GitVersion,
		serverVersion.GitCommit,
		serverVersion.BuildDate,
		serverVersion.GoVersion,
		serverVersion.Platform,
	), nil
}
