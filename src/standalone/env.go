package standalone

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/pkg/errors"
)

type containerInfo struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type environment struct {
	Mode         InstallMode `json:"mode"`
	CanFail      bool        `json:"canFail"`
	InstallerUrl string      `json:"installerUrl"`

	InstallerFlavor string          `json:"installerFlavor"`
	InstallerTech   []string        `json:"installerTech"`
	InstallPath     string          `json:"installPath"`
	Containers      []containerInfo `json:"containers"`

	K8NodeName    string `json:"k8NodeName"`
	K8PodName     string `json:"k8PodName"`
	K8PodUID      string `json:"k8BasePodUID"`
	K8ClusterID   string `json:"k8ClusterID"`
	K8BasePodName string `json:"k8BasePodName"`
	K8Namespace   string `json:"k8Namespace"`

	WorkloadKind string `json:"workloadKind"`
	WorkloadName string `json:"workloadName"`

	OneAgentInjected   bool `json:"oneAgentInjected"`
	DataIngestInjected bool `json:"dataIngestInjected"`
}

func newEnv() (*environment, error) {
	log.Info("checking envvars")
	env := &environment{}
	env.setOptionalFields()
	err := env.setRequiredFields()
	if err != nil {
		return nil, err
	}
	log.Info("envvars checked", "env", env)
	return env, nil
}

func (env *environment) setRequiredFields() error {
	errs := []error{}
	requiredFieldSetters := []func() error{
		env.addCanFail,
	}
	if env.OneAgentInjected {
		oneAgentFieldSetters := []func() error{
			env.addMode,
			env.addInstallerTech,
			env.addInstallPath,
			env.addContainers,
			env.addK8NodeName,
			env.addK8PodName,
			env.addK8PodUID,
			env.addK8BasePodName,
			env.addK8Namespace,
		}
		requiredFieldSetters = append(requiredFieldSetters, oneAgentFieldSetters...)
	}

	if env.DataIngestInjected {
		dataIngestFieldSetters := []func() error{
			env.addWorkloadKind,
			env.addWorkloadName,
			env.addK8ClusterID,
		}
		requiredFieldSetters = append(requiredFieldSetters, dataIngestFieldSetters...)
	}

	for _, setField := range requiredFieldSetters {
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
	env.addInstallerUrl()
	env.addInstallerFlavor()
	env.addOneAgentInjected()
	env.addDataIngestInjected()
}

func (env *environment) addMode() error {
	mode, err := checkEnvVar(ModeEnv)
	if err != nil {
		return err
	}
	env.Mode = InstallMode(mode)
	return nil
}

func (env *environment) addCanFail() error {
	canFail, err := checkEnvVar(CanFailEnv)
	if err != nil {
		return err
	}
	env.CanFail = canFail == "fail"
	return nil
}

func (env *environment) addInstallerFlavor() {
	flavor, _ := checkEnvVar(InstallerFlavorEnv)
	if flavor == "" {
		env.InstallerFlavor = arch.Flavor
	} else {
		env.InstallerFlavor = flavor
	}
}

func (env *environment) addInstallerTech() error {
	technologies, err := checkEnvVar(InstallerTechEnv)
	if err != nil {
		return err
	}
	env.InstallerTech = strings.Split(technologies, ",")
	return nil
}

func (env *environment) addInstallPath() error {
	installPath, err := checkEnvVar(InstallPathEnv)
	if err != nil {
		return err
	}
	env.InstallPath = installPath
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

		containerName, err := checkEnvVar(nameEnv)
		if err != nil {
			return err
		}
		imageName, err := checkEnvVar(imageEnv)
		if err != nil {
			return err
		}
		containers = append(containers, containerInfo{
			Name:  containerName,
			Image: imageName,
		})
	}
	env.Containers = containers
	return nil
}

func (env *environment) addK8NodeName() error {
	nodeName, err := checkEnvVar(K8NodeNameEnv)
	if err != nil {
		return err
	}
	env.K8NodeName = nodeName
	return nil
}

func (env *environment) addK8PodName() error {
	podName, err := checkEnvVar(K8PodNameEnv)
	if err != nil {
		return err
	}
	env.K8PodName = podName
	return nil
}

func (env *environment) addK8PodUID() error {
	podUID, err := checkEnvVar(K8PodUIDEnv)
	if err != nil {
		return err
	}
	env.K8PodUID = podUID
	return nil
}

func (env *environment) addK8ClusterID() error {
	clusterID, err := checkEnvVar(K8ClusterIDEnv)
	if err != nil {
		return err
	}
	env.K8ClusterID = clusterID
	return nil
}

func (env *environment) addK8BasePodName() error {
	basePodName, err := checkEnvVar(K8BasePodNameEnv)
	if err != nil {
		return err
	}
	env.K8BasePodName = basePodName
	return nil
}

func (env *environment) addK8Namespace() error {
	namespace, err := checkEnvVar(K8NamespaceEnv)
	if err != nil {
		return err
	}
	env.K8Namespace = namespace
	return nil
}

func (env *environment) addWorkloadKind() error {
	workloadKind, err := checkEnvVar(WorkloadKindEnv)
	if err != nil {
		return err
	}
	env.WorkloadKind = workloadKind
	return nil
}

func (env *environment) addWorkloadName() error {
	workloadName, err := checkEnvVar(WorkloadNameEnv)
	if err != nil {
		return err
	}
	env.WorkloadName = workloadName
	return nil
}

func (env *environment) addInstallerUrl() {
	url, _ := checkEnvVar(InstallerUrlEnv)
	env.InstallerUrl = url
}

func (env *environment) addOneAgentInjected() {
	oneAgentInjected, _ := checkEnvVar(OneAgentInjectedEnv)
	env.OneAgentInjected = oneAgentInjected == "true"
}

func (env *environment) addDataIngestInjected() {
	dataIngestInjected, _ := checkEnvVar(DataIngestInjectedEnv)
	env.DataIngestInjected = dataIngestInjected == "true"
}

func checkEnvVar(envvar string) (string, error) {
	result := os.Getenv(envvar)
	if result == "" {
		return "", errors.Errorf("%s environment variable is missing", envvar)
	}
	return result, nil
}
