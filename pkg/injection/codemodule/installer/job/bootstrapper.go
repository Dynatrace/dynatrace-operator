package job

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8senv"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8slabel"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/objects/k8sjob"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	webhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
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
	appLabels := k8slabel.NewAppLabels(k8slabel.CodeModuleComponentLabel, inst.props.Owner.GetName(), "", "")

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
				Path: k8senv.GetCSIDataDir(),
			},
		},
	}

	container.Args = inst.buildArgs(name, targetDir)

	annotations := maputils.MergeMap(inst.props.CSIJob.Annotations, map[string]string{
		webhook.AnnotationDynatraceInject: "false",
	})

	return k8sjob.Build(inst.props.Owner, name, container,
		k8sjob.SetAnnotations(inst.props.CSIJob.Annotations),
		k8sjob.SetPodAnnotations(annotations),
		k8sjob.SetNodeName(inst.nodeName),
		k8sjob.SetPullSecret(inst.props.PullSecrets...),
		k8sjob.SetTolerations(inst.props.CSIJob.Tolerations),
		k8sjob.SetAllLabels(appLabels.BuildLabels(), map[string]string{}, appLabels.BuildLabels(), inst.props.CSIJob.Labels),
		k8sjob.AddLabels(inst.props.CSIJob.Labels),
		k8sjob.SetVolumes([]corev1.Volume{hostVolume}),
		k8sjob.SetOnFailureRestartPolicy(),
		k8sjob.SetAutomountServiceAccountToken(false),
		k8sjob.SetActiveDeadlineSeconds(activeDeadlineSeconds),
		k8sjob.SetTTLSecondsAfterFinished(ttlSecondsAfterFinished),
		k8sjob.SetServiceAccount(provisionerServiceAccount),
		k8sjob.SetPriorityClassName(inst.props.CSIJob.Job.PriorityClassName),
	)
}

func (inst *Installer) buildArgs(jobName, targetDir string) []string {
	return []string{
		"--source=" + codeModuleSource,
		"--target=" + targetDir,
		"--work=" + inst.props.PathResolver.AgentJobWorkDirForJob(jobName),
	}
}
