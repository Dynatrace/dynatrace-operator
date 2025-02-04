package startup

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/Dynatrace/dynatrace-operator/pkg/arch"
	"github.com/Dynatrace/dynatrace-operator/pkg/consts"
	"github.com/pkg/errors"
)

const (
	trueStatement = "true"
	silentPhrase  = "silent"
	failPhrase    = "fail"
)

type ContainerInfo struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type environment struct {
	FailurePolicy string `json:"failurePolicy"`
	InstallerUrl  string `json:"installerUrl"`

	InstallerFlavor string `json:"installerFlavor"`
	InstallVersion  string `json:"installVersion"`
	InstallPath     string `json:"installPath"`

	K8NodeName        string `json:"k8NodeName"`
	K8PodName         string `json:"k8PodName"`
	K8PodUID          string `json:"k8BasePodUID"`
	K8BasePodName     string `json:"k8BasePodName"`
	K8Namespace       string `json:"k8Namespace"`
	K8ClusterID       string `json:"k8ClusterID"`
	K8ClusterName     string `json:"k8sClusterName"`
	K8ClusterEntityID string `json:"k8sClusterEntityID"`

	WorkloadKind        string            `json:"workloadKind"`
	WorkloadName        string            `json:"workloadName"`
	WorkloadAnnotations map[string]string `json:"workloadAnnotations"`

	InstallerTech []string        `json:"installerTech"`
	Containers    []ContainerInfo `json:"containers"`

	OneAgentInjected           bool `json:"oneAgentInjected"`
	MetadataEnrichmentInjected bool `json:"metadataEnrichmentInjected"`
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

	requiredFieldSetters := env.getCommonFieldSetters()

	if env.OneAgentInjected {
		requiredFieldSetters = append(requiredFieldSetters, env.getOneAgentFieldSetters()...)
	}

	if env.MetadataEnrichmentInjected {
		requiredFieldSetters = append(requiredFieldSetters, env.getMetadataEnrichmentFieldSetters()...)
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
		env.addContainers,
		env.addK8PodName,
		env.addK8PodUID,
		env.addK8Namespace,
		env.addK8ClusterID,
		env.addK8NodeName,
		env.addFailurePolicy,
	}
}

func (env *environment) getOneAgentFieldSetters() []func() error {
	return []func() error{
		env.addInstallerTech,
		env.addInstallPath,
		env.addK8BasePodName,
	}
}

func (env *environment) getMetadataEnrichmentFieldSetters() []func() error {
	return []func() error{
		env.addWorkloadKind,
		env.addWorkloadName,
		env.addWorkloadAnnotations,
	}
}

func (env *environment) setOptionalFields() {
	env.addInstallerUrl()
	env.addInstallerFlavor()
	env.addInstallVersion()
	env.addClusterName()
	env.addEntityID()
}

func (env *environment) setMutationTypeFields() {
	env.addOneAgentInjected()
	env.addMetadataEnrichmentInjected()
}

func (env *environment) addFailurePolicy() error {
	failurePolicy, err := checkEnvVar(consts.InjectionFailurePolicyEnv)
	if err != nil {
		return err
	}

	switch failurePolicy {
	case failPhrase:
		env.FailurePolicy = failPhrase
	default:
		env.FailurePolicy = silentPhrase
	}

	return nil
}

func (env *environment) addInstallerFlavor() {
	flavor, _ := checkEnvVar(consts.AgentInstallerFlavorEnv)
	if flavor == "" {
		env.InstallerFlavor = arch.Flavor
	} else {
		env.InstallerFlavor = flavor
	}
}

func (env *environment) addInstallerTech() error {
	technologies, err := checkEnvVar(consts.AgentInstallerTechEnv)
	if err != nil {
		return err
	}

	env.InstallerTech = strings.Split(technologies, ",")

	return nil
}

func (env *environment) addInstallPath() error {
	installPath, err := checkEnvVar(consts.AgentInstallPathEnv)
	if err != nil {
		return err
	}

	env.InstallPath = installPath

	return nil
}

func (env *environment) addContainers() error {
	rawContainers, err := checkEnvVar(consts.ContainerInfoEnv)
	if err != nil {
		return err
	}

	var containers []ContainerInfo

	err = json.Unmarshal([]byte(rawContainers), &containers)
	if err != nil {
		return err
	}

	env.Containers = containers

	return nil
}

func (env *environment) addK8NodeName() error {
	nodeName, err := checkEnvVar(consts.K8sNodeNameEnv)
	if err != nil {
		return err
	}

	env.K8NodeName = nodeName

	return nil
}

func (env *environment) addK8PodName() error {
	podName, err := checkEnvVar(consts.K8sPodNameEnv)
	if err != nil {
		return err
	}

	env.K8PodName = podName

	return nil
}

func (env *environment) addK8PodUID() error {
	podUID, err := checkEnvVar(consts.K8sPodUIDEnv)
	if err != nil {
		return err
	}

	env.K8PodUID = podUID

	return nil
}

func (env *environment) addK8BasePodName() error {
	basePodName, err := checkEnvVar(consts.K8sBasePodNameEnv)
	if err != nil {
		return err
	}

	env.K8BasePodName = basePodName

	return nil
}

func (env *environment) addK8Namespace() error {
	namespace, err := checkEnvVar(consts.K8sNamespaceEnv)
	if err != nil {
		return err
	}

	env.K8Namespace = namespace

	return nil
}

func (env *environment) addWorkloadKind() error {
	workloadKind, err := checkEnvVar(consts.EnrichmentWorkloadKindEnv)
	if err != nil {
		return err
	}

	env.WorkloadKind = workloadKind

	return nil
}

func (env *environment) addWorkloadName() error {
	workloadName, err := checkEnvVar(consts.EnrichmentWorkloadNameEnv)
	if err != nil {
		return err
	}

	env.WorkloadName = workloadName

	return nil
}

func (env *environment) addWorkloadAnnotations() error {
	workloadAnnotationsJson, err := checkEnvVar(consts.EnrichmentWorkloadAnnotationsEnv)
	if err != nil {
		return err
	}

	workloadAnnotations := map[string]string{}
	err = json.Unmarshal([]byte(workloadAnnotationsJson), &workloadAnnotations)

	if err != nil {
		return err
	}

	env.WorkloadAnnotations = workloadAnnotations

	return nil
}

func (env *environment) addClusterName() {
	clusterName, _ := checkEnvVar(consts.EnrichmentClusterNameEnv)
	env.K8ClusterName = clusterName
}

func (env *environment) addEntityID() {
	entityID, _ := checkEnvVar(consts.EnrichmentClusterEntityIDEnv)
	env.K8ClusterEntityID = entityID
}

func (env *environment) addInstallerUrl() {
	url, _ := checkEnvVar(consts.AgentInstallerUrlEnv)
	env.InstallerUrl = url
}

func (env *environment) addInstallVersion() {
	version, _ := checkEnvVar(consts.AgentInstallerVersionEnv)
	env.InstallVersion = version
}

func (env *environment) addOneAgentInjected() {
	oneAgentInjected, _ := checkEnvVar(consts.AgentInjectedEnv)
	env.OneAgentInjected = oneAgentInjected == trueStatement
}

func (env *environment) addMetadataEnrichmentInjected() {
	metadataEnrichmentInjected, _ := checkEnvVar(consts.EnrichmentInjectedEnv)
	env.MetadataEnrichmentInjected = metadataEnrichmentInjected == trueStatement
}

func (env *environment) addK8ClusterID() error {
	clusterID, err := checkEnvVar(consts.K8sClusterIDEnv)
	if err != nil {
		return err
	}

	env.K8ClusterID = clusterID

	return nil
}

func checkEnvVar(envvar string) (string, error) {
	result := os.Getenv(envvar)
	if result == "" {
		return "", errors.Errorf("%s environment variable is missing", envvar)
	}

	return result, nil
}
