package job

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/csi/metadata"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	jobutil "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/job"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	namePrefix = "codemodule-download-"

	volumeName = "dynatrace-codemodules"
	mountPath  = "/mnt/bins"

	codeModuleSource = "/opt/dynatrace/oneagent"
)

func (inst *Installer) buildJobName() string {
	hashPostfix, _ := hasher.GenerateHash(inst.props.ImageUri + inst.nodeName)

	return namePrefix + hashPostfix
}

func (inst *Installer) buildJob(name string) (*batchv1.Job, error) {
	tolerations, err := env.GetTolerations()
	if err != nil {
		log.Info("failed to get tolerations from env")

		return nil, err
	}

	appLabels := labels.NewAppLabels(labels.CodeModuleComponentLabel, inst.props.Owner.GetName(), "", "")

	container := corev1.Container{
		Name:            "codemodule-download",
		Image:           inst.props.ImageUri,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/opt/dynatrace/bin/bootstrap"},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: mountPath,
			},
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:              ptr.To(int64(0)),
			ReadOnlyRootFilesystem: ptr.To(true),
			SELinuxOptions: &corev1.SELinuxOptions{
				Level: "s0",
			},
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}

	hostVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: env.GetCSIDataDir(),
			},
		},
	}

	container.Args = inst.buildArgs(name)

	return jobutil.Build(inst.props.Owner, name, container,
		jobutil.SetNodeName(inst.nodeName),
		jobutil.SetPullSecret(inst.props.PullSecrets...),
		jobutil.SetTolerations(tolerations),
		jobutil.SetAllLabels(appLabels.BuildLabels(), map[string]string{}, appLabels.BuildLabels(), map[string]string{}),
		jobutil.SetVolumes([]corev1.Volume{hostVolume}),
		jobutil.SetOnFailureRestartPolicy(),
		jobutil.SetAutomountServiceAccountToken(false),
	)
}

func (inst *Installer) buildArgs(jobName string) []string {
	path := metadata.PathResolver{RootDir: mountPath}

	return []string{
		"--source=" + codeModuleSource,
		"--target=" + path.AgentSharedBinaryDirForAgent(inst.props.Version),
		"--work=" + path.AgentJobWorkDirForJob(jobName),
	}
}
