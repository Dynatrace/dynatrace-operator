package standalone

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/pkg/errors"
)

type InstallMode string

const (
	InstallerMode InstallMode = "installer"
	CsiMode       InstallMode = "csi"

	ModeEnv         = "MODE"
	CanFailEnv      = "FAIL_POLICY"
	InstallerUrlEnv = "INSTALLER_URL"

	InstallerFlavorEnv = "FLAVOR"
	InstallerTechEnv   = "TECHNOLOGIES"
	InstallerArchEnv   = "ARCH"

	K8NodeNameEnv    = "K8S_NODE_NAME"
	K8PodNameEnv     = "K8S_PODNAME"
	K8PodUIDEnv      = "K8S_PODUID"
	K8BasePodNameEnv = "K8S_BASEPODNAME"
	K8NamespaceEnv   = "K8S_NAMESPACE"

	WorkloadKindEnv = "DT_WORKLOAD_KIND"
	WorkloadNameEnv = "DT_WORKLOAD_NAME"

	InstallPathEnv            = "INSTALLPATH"
	ContainerCountEnv         = "CONTAINER_COUNT"
	ContainerNameEnvTemplate  = "CONTAINER_%d_NAME"
	ContainerImageEnvTemplate = "CONTAINER_%d_IMAGE"

	OneAgentInjectedEnv   = "ONEAGENT_INJECTED"
	DataIngestInjectedEnv = "DATA_INGEST_INJECTED"
)

type containerInfo struct {
	name  string
	image string
}

type environment struct {
	mode         InstallMode
	canFail      bool
	installerUrl string

	installerFlavor string
	installerTech   []string
	installerArch   string
	installPath     string
	containers      []containerInfo

	k8NodeName    string
	k8PodName     string
	k8PodUID      string
	k8BasePodName string
	k8Namespace   string

	workloadKind string
	workloadName string

	oneAgentInjected   bool
	dataIngestInjected bool
}

func newEnv() (*environment, error) {
	env := &environment{}
	err := env.setRequiredFields()
	if err != nil {
		return nil, err
	}
	env.setOptionalFields()
	return env, nil
}

func (env *environment) setRequiredFields() error {
	errs := []error{}
	fieldSetters := []func() error{
		env.addMode,
		env.addCanFail,
		env.addInstallerFlavor,
		env.addInstallerTech,
		env.addInstallPath,
		env.addContainers,
		env.addK8NodeName,
		env.addK8PodName,
		env.addK8PodUID,
		env.addK8BasePodName,
		env.addK8Namespace,
		env.addOneAgentInjected,
		env.addDataIngestInjected,
	}
	for _, setField := range fieldSetters {
		if err := setField(); err != nil {
			errs = append(errs, err)
			log.Info(err.Error())
		}
	}
	if len(errs) != 0 {
		return errors.Errorf("%d envvars missing", len(errs))
	}
	return nil
}

func (env *environment) setOptionalFields() {
	env.addWorkloadKind()
	env.addWorkloadName()
	env.addInstallerUrl()
	env.addInstallerArch()
}

func (env *environment) addMode() error {
	mode, err := checkEnvVar(ModeEnv)
	if err != nil {
		return err
	}
	env.mode = InstallMode(mode)
	return nil
}

func (env *environment) addCanFail() error {
	canFail, err := checkEnvVar(CanFailEnv)
	if err != nil {
		return err
	}
	if canFail == "true" {
		env.canFail = true
	} else {
		env.canFail = false
	}
	return nil
}

func (env *environment) addInstallerFlavor() error {
	flavor, err := checkEnvVar(InstallerFlavorEnv)
	if err != nil {
		return err
	}
	env.installerFlavor = flavor
	return nil
}

func (env *environment) addInstallerTech() error {
	technologies, err := checkEnvVar(InstallerTechEnv)
	if err != nil {
		return err
	}
	env.installerTech = strings.Split(technologies, ",")
	return nil
}

func (env *environment) addInstallerArch() {
	arch, err := checkEnvVar(InstallerArchEnv)
	if err != nil {
		env.installerArch = dtclient.ArchX86
	} else {
		env.installerArch = arch
	}

}

func (env *environment) addInstallPath() error {
	installPath, err := checkEnvVar(InstallPathEnv)
	if err != nil {
		return err
	}
	env.installPath = installPath
	return nil
}

func (env *environment) addContainers() error {
	containers := []containerInfo{}
	containerCountStr, err := checkEnvVar(ContainerCountEnv)
	if err != nil {
		return err
	}
	countCount, err := strconv.Atoi(containerCountStr)
	if err != nil {
		return err
	}
	for i := 1; i <= countCount; i++ {
		nameEnv := fmt.Sprintf(ContainerNameEnvTemplate, i)
		imageEnv := fmt.Sprintf(ContainerImageEnvTemplate, i)

		containeName, err := checkEnvVar(nameEnv)
		if err != nil {
			return err
		}
		imageName, err := checkEnvVar(imageEnv)
		if err != nil {
			return err
		}
		containers = append(containers, containerInfo{
			name:  containeName,
			image: imageName,
		})
	}
	env.containers = containers
	return nil
}

func (env *environment) addK8NodeName() error {
	nodeName, err := checkEnvVar(K8NodeNameEnv)
	if err != nil {
		return err
	}
	env.k8NodeName = nodeName
	return nil
}

func (env *environment) addK8PodName() error {
	podName, err := checkEnvVar(K8PodNameEnv)
	if err != nil {
		return err
	}
	env.k8PodName = podName
	return nil
}

func (env *environment) addK8PodUID() error {
	podUID, err := checkEnvVar(K8PodUIDEnv)
	if err != nil {
		return err
	}
	env.k8PodUID = podUID
	return nil
}

func (env *environment) addK8BasePodName() error {
	basePodName, err := checkEnvVar(K8BasePodNameEnv)
	if err != nil {
		return err
	}
	env.k8BasePodName = basePodName
	return nil
}

func (env *environment) addK8Namespace() error {
	namespace, err := checkEnvVar(K8NamespaceEnv)
	if err != nil {
		return err
	}
	env.k8Namespace = namespace
	return nil
}

func (env *environment) addWorkloadKind() {
	workloadKind, _ := checkEnvVar(WorkloadKindEnv)
	env.workloadKind = workloadKind
}

func (env *environment) addWorkloadName() {
	workloadName, _ := checkEnvVar(WorkloadNameEnv)
	env.workloadName = workloadName
}

func (env *environment) addInstallerUrl() {
	url, _ := checkEnvVar(InstallerUrlEnv)
	env.installerUrl = url
}

func (env *environment) addOneAgentInjected() error {
	oneAgentInjected, err := checkEnvVar(OneAgentInjectedEnv)
	if err != nil {
		return err
	}
	if oneAgentInjected == "true" {
		env.oneAgentInjected = true
	} else {
		env.oneAgentInjected = false
	}
	return nil
}

func (env *environment) addDataIngestInjected() error {
	dataIngestInjected, err := checkEnvVar(DataIngestInjectedEnv)
	if err != nil {
		return err
	}
	if dataIngestInjected == "true" {
		env.dataIngestInjected = true
	} else {
		env.dataIngestInjected = false
	}
	return nil
}

func checkEnvVar(envvar string) (string, error) {
	result := os.Getenv(envvar)
	if result == "" {
		return "", errors.Errorf("%s environment variable is missing", envvar)
	}
	return result, nil
}
