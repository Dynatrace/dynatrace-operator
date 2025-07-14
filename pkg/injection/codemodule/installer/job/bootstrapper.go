package job

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/env"
	jobutil "github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/job"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/common"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	namePrefix = "codemodule-download-"

	volumeName       = "dynatrace-codemodules"
	codeModuleSource = "/opt/dynatrace/oneagent"

	provisionerServiceAccount = "dynatrace-oneagent-csi-driver" // TODO: Get it from env

	activeDeadlineSeconds   int64 = 600 // 10 min, after which the Job will be put into a Failed state
	ttlSecondsAfterFinished int32 = 10  // 10 sec after the Job is put into a Succeeded or Failed state the Job and related Pods will be terminated
)

func (inst *Installer) buildJobName() string {
	hashPostfix, _ := hasher.GenerateHash(inst.props.ImageURI + inst.nodeName)

	return namePrefix + hashPostfix
}

func (inst *Installer) buildJob(name, targetDir string) (*batchv1.Job, error) {
	appLabels := labels.NewAppLabels(labels.CodeModuleComponentLabel, inst.props.Owner.GetName(), "", "")

	container := corev1.Container{
		Name:            "codemodule-download",
		Image:           inst.props.ImageURI,
		ImagePullPolicy: corev1.PullIfNotPresent,
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      volumeName,
				MountPath: inst.props.PathResolver.RootDir,
			},
		},
		SecurityContext: &inst.props.CSIJob.Job.SecurityContext,
		Resources:       inst.props.CSIJob.Job.Resources,
	}

	hostVolume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: env.GetCSIDataDir(),
			},
		},
	}

	container.Args = inst.buildArgs(name, targetDir)

	annotations := maputils.MergeMap(inst.props.CSIJob.Annotations, map[string]string{
		webhook.AnnotationDynatraceInject: "false",
	})

	return jobutil.Build(inst.props.Owner, name, container,
		jobutil.SetAnnotations(inst.props.CSIJob.Annotations),
		jobutil.SetPodAnnotations(annotations),
		jobutil.SetNodeName(inst.nodeName),
		jobutil.SetPullSecret(inst.props.PullSecrets...),
		jobutil.SetTolerations(inst.props.CSIJob.Tolerations),
		jobutil.SetAllLabels(appLabels.BuildLabels(), map[string]string{}, appLabels.BuildLabels(), inst.props.CSIJob.Labels),
		jobutil.AddLabels(inst.props.CSIJob.Labels),
		jobutil.SetVolumes([]corev1.Volume{hostVolume}),
		jobutil.SetOnFailureRestartPolicy(),
		jobutil.SetAutomountServiceAccountToken(false),
		jobutil.SetActiveDeadlineSeconds(activeDeadlineSeconds),
		jobutil.SetTTLSecondsAfterFinished(ttlSecondsAfterFinished),
		jobutil.SetServiceAccount(provisionerServiceAccount),
	)
}

func (inst *Installer) buildArgs(jobName, targetDir string) []string {
	return []string{
		"--source=" + codeModuleSource,
		"--target=" + targetDir,
		"--work=" + inst.props.PathResolver.AgentJobWorkDirForJob(jobName),
	}
}
