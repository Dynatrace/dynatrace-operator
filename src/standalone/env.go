package standalone

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/src/arch"
	"github.com/Dynatrace/dynatrace-operator/src/config"
	"github.com/pkg/errors"
)

const (
	trueStatement = "true"
	silentPhrase  = "silent"
	failPhrase    = "fail"
	forcePhrase   = "force"
)

type containerInfo struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type environment struct {
	Mode          config.InstallMode `json:"mode"`
	FailurePolicy string             `json:"failurePolicy"`
	InstallerUrl  string             `json:"installerUrl"`

	InstallerFlavor string          `json:"installerFlavor"`
	InstallVersion  string          `json:"installVersion"`
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
	IsReadOnlyCSI      bool `json:"isReadOnlyCSI"`
}

func newEnv() (*environment, error) {
	log.Info("checking envvars")
	env := &environment{}
	env.setMutationTypeFields()
	err := env.setRequiredFields()
	if err != nil {
		return nil, err
	}
	env.setOptionalFields()
	log.Info("envvars checked", "env", env)
	return env, nil
}

func (env *environment) setRequiredFields() error {
	errs := []error{}
	requiredFieldSetters := []func() error{
		env.addFailurePolicy,
	}
	if env.OneAgentInjected {
		requiredFieldSetters = append(requiredFieldSetters, env.getOneAgentFieldSetters()...)
	}

	if env.DataIngestInjected {
		requiredFieldSetters = append(requiredFieldSetters, env.getDataIngestFieldSetters()...)
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

func (env *environment) getCommonFieldSetters() []func() error {
	return []func() error{
		env.addK8PodName,
		env.addK8PodUID,
		env.addK8Namespace,
	}
}

func (env *environment) getOneAgentFieldSetters() []func() error {
	return append(env.getCommonFieldSetters(),
		env.addMode,
		env.addInstallerTech,
		env.addInstallPath,
		env.addContainers,
		env.addK8NodeName,
		env.addK8BasePodName,
	)
}

func (env *environment) getDataIngestFieldSetters() []func() error {
	return append(env.getCommonFieldSetters(),
		env.addWorkloadKind,
		env.addWorkloadName,
		env.addK8ClusterID,
	)
}

func (env *environment) setOptionalFields() {
	env.addInstallerUrl()
	env.addInstallerFlavor()
	env.addInstallVersion()
}

func (env *environment) setMutationTypeFields() {
	env.addOneAgentInjected()
	env.addDataIngestInjected()
	env.addIsReadOnlyCSI()
}

func (env *environment) addMode() error {
	mode, err := checkEnvVar(config.AgentInstallModeEnv)
	if err != nil {
		return err
	}
	env.Mode = config.InstallMode(mode)
	return nil
}

func (env *environment) addFailurePolicy() error {
	failurePolicy, err := checkEnvVar(config.InjectionFailurePolicyEnv)
	if err != nil {
		return err
	}
	switch failurePolicy {
	case failPhrase:
		env.FailurePolicy = failPhrase
	case forcePhrase:
		env.FailurePolicy = forcePhrase
	default:
		env.FailurePolicy = silentPhrase
	}
	return nil
}

func (env *environment) addInstallerFlavor() {
	flavor, _ := checkEnvVar(config.AgentInstallerFlavorEnv)
	if flavor == "" {
		env.InstallerFlavor = arch.Flavor
	} else {
		env.InstallerFlavor = flavor
	}
}

func (env *environment) addInstallerTech() error {
	technologies, err := checkEnvVar(config.AgentInstallerTechEnv)
	if err != nil {
		return err
	}
	env.InstallerTech = strings.Split(technologies, ",")
	return nil
}

func (env *environment) addInstallPath() error {
	installPath, err := checkEnvVar(config.AgentInstallPathEnv)
	if err != nil {
		return err
	}
	env.InstallPath = installPath
	return nil
}

func (env *environment) addContainers() error {
	containers := []containerInfo{}
	containerCountStr, err := checkEnvVar(config.AgentContainerCountEnv)
	if err != nil {
		return err
	}
	countCount, err := strconv.Atoi(containerCountStr)
	if err != nil {
		return err
	}
	for i := 1; i <= countCount; i++ {
		nameEnv := fmt.Sprintf(config.AgentContainerNameEnvTemplate, i)
		imageEnv := fmt.Sprintf(config.AgentContainerImageEnvTemplate, i)

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
	nodeName, err := checkEnvVar(config.K8sNodeNameEnv)
	if err != nil {
		return err
	}
	env.K8NodeName = nodeName
	return nil
}

func (env *environment) addK8PodName() error {
	podName, err := checkEnvVar(config.K8sPodNameEnv)
	if err != nil {
		return err
	}
	env.K8PodName = podName
	return nil
}

func (env *environment) addK8PodUID() error {
	podUID, err := checkEnvVar(config.K8sPodUIDEnv)
	if err != nil {
		return err
	}
	env.K8PodUID = podUID
	return nil
}

func (env *environment) addK8ClusterID() error {
	clusterID, err := checkEnvVar(config.K8sClusterIDEnv)
	if err != nil {
		return err
	}
	env.K8ClusterID = clusterID
	return nil
}

func (env *environment) addK8BasePodName() error {
	basePodName, err := checkEnvVar(config.K8sBasePodNameEnv)
	if err != nil {
		return err
	}
	env.K8BasePodName = basePodName
	return nil
}

func (env *environment) addK8Namespace() error {
	namespace, err := checkEnvVar(config.K8sNamespaceEnv)
	if err != nil {
		return err
	}
	env.K8Namespace = namespace
	return nil
}

func (env *environment) addWorkloadKind() error {
	workloadKind, err := checkEnvVar(config.EnrichmentWorkloadKindEnv)
	if err != nil {
		return err
	}
	if workloadKind == config.EnrichmentUnknownWorkload {
		env.WorkloadKind = ""
	} else {
		env.WorkloadKind = workloadKind
	}
	return nil
}

func (env *environment) addWorkloadName() error {
	workloadName, err := checkEnvVar(config.EnrichmentWorkloadNameEnv)
	if err != nil {
		return err
	}
	if workloadName == config.EnrichmentUnknownWorkload {
		env.WorkloadName = ""
	} else {
		env.WorkloadName = workloadName
	}
	return nil
}

func (env *environment) addInstallerUrl() {
	url, _ := checkEnvVar(config.AgentInstallerUrlEnv)
	env.InstallerUrl = url
}

func (env *environment) addInstallVersion() {
	version, _ := checkEnvVar(config.AgentInstallerVersionEnv)
	env.InstallVersion = version
}

func (env *environment) addOneAgentInjected() {
	oneAgentInjected, _ := checkEnvVar(config.AgentInjectedEnv)
	env.OneAgentInjected = oneAgentInjected == trueStatement
}

func (env *environment) addIsReadOnlyCSI() {
	isReadOnlyCSI, _ := checkEnvVar(config.AgentReadonlyCSI)
	env.IsReadOnlyCSI = isReadOnlyCSI == trueStatement
}

func (env *environment) addDataIngestInjected() {
	dataIngestInjected, _ := checkEnvVar(config.EnrichmentInjectedEnv)
	env.DataIngestInjected = dataIngestInjected == trueStatement
}

func checkEnvVar(envvar string) (string, error) {
	result := os.Getenv(envvar)
	if result == "" {
		return "", errors.Errorf("%s environment variable is missing", envvar)
	}
	return result, nil
}
