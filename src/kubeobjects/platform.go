package kubeobjects

import "os"

const (
	platformEnvName            = "PLATFORM"
	openshiftPlatformEnvValue  = "openshift"
	kubernetesPlatformEnvValue = "kubernetes"
)

type Platform int

const (
	Kubernetes Platform = iota
	Openshift
)

func ResolvePlatformFromEnv() Platform {
	switch os.Getenv(platformEnvName) {
	case openshiftPlatformEnvValue:
		return Openshift
	case kubernetesPlatformEnvValue:
		fallthrough
	default:
		return Kubernetes
	}
}

func GetPlatformFromEnv() string {
	switch os.Getenv(platformEnvName) {
	case openshiftPlatformEnvValue:
		return openshiftPlatformEnvValue
	default:
		return kubernetesPlatformEnvValue
	}
}
