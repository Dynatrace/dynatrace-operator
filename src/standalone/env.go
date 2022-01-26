package standalone

import (
	"os"

	"github.com/pkg/errors"
)

type installMode string

const (
	installerMode installMode = "installer"
	csiMode       installMode = "csi"

	ModeEnv    = "MODE"
	CanFailEnv = "FAIL_POLICY"

	InstallerFlavorEnv = "FLAVOR"
	InstallerTechEnv   = "TECHNOLOGIES"

	K8NodeNameEnv    = "K8_NODENAME"
	K8PodNameEnv     = "K8_PODNAME"
	K8PodUIDEnv      = "K8_PODUID"
	K8BasePodNameEnv = "K8_BASEPODNAME"
	K8NamespaceEnv   = "K8_NAMESPACE"

	WorkloadKindEnv = "DT_WORKLOAD_KIND"
	WorkloadNameEnv = "DT_WORKLOAD_NAME"
)

type environment struct {
	mode    installMode
	canFail bool

	installerFlavor string
	installerTech   string

	k8NodeName    string
	k8PodName     string
	k8PodUID      string
	k8BasePodName string
	k8Namespace   string

	workloadKind string
	workloadName string
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

	if err := env.addMode(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addCanFail(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addInstallerFlavor(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addInstallerTech(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addK8NodeName(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addK8PodName(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addK8PodUID(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addK8BasePodName(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if err := env.addK8Namespace(); err != nil {
		errs = append(errs, err)
		log.Info(err.Error())
	}
	if len(errs) != 0 {
		return errors.Errorf("%d envvars missing", len(errs))
	}
	return nil
}

func (env *environment) setOptionalFields() {
	env.addWorkloadKind()
	env.addWorkloadName()
}

func (env *environment) addMode() error {
	mode, err := checkEnvVar(ModeEnv)
	if err != nil {
		return err
	}
	env.mode = installMode(mode)
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
	env.installerTech = technologies
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

func checkEnvVar(envvar string) (string, error) {
	result := os.Getenv(envvar)
	if result == "" {
		return "", errors.Errorf("%s environment variable is missing", envvar)
	}
	return result, nil
}
