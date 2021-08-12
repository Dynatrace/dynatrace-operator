package dtcsi

import (
	"context"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	RegistrarDirPath    = "/var/lib/kubelet/plugins_registry/"
	PluginDirPath       = "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com"
	PluginsDirPath      = "/var/lib/kubelet/plugins"
	MountpointDirPath   = "/var/lib/kubelet/pods"
	OneAgentDataDirPath = "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com/data"
)

type Reconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	logger            logr.Logger
	instance          *v1alpha1.DynaKube
	operatorPodName   string
	operatorNamespace string
}

func NewReconciler(
	client client.Client, scheme *runtime.Scheme, logger logr.Logger,
	instance *v1alpha1.DynaKube, operatorPodName, operatorNamespace string) *Reconciler {
	return &Reconciler{
		client:            client,
		scheme:            scheme,
		logger:            logger,
		instance:          instance,
		operatorPodName:   operatorPodName,
		operatorNamespace: operatorNamespace,
	}
}

func (r *Reconciler) Reconcile() (bool, error) {
	r.logger.Info("Reconciling CSI driver")

	operatorImage, err := r.getOperatorImage()
	if err != nil {
		return false, errors.WithStack(err)
	}

	ds, err := buildDesiredCSIDaemonSet(operatorImage, r.operatorNamespace, r.instance.Spec.CodeModules.ServiceAccountNameCSIDriver)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if err := controllerutil.SetControllerReference(r.instance, ds, r.scheme); err != nil {
		return false, errors.WithStack(err)
	}

	upd, err := kubeobjects.CreateOrUpdateDaemonSet(r.client, r.logger, ds)
	if upd || err != nil {
		return upd, errors.WithStack(err)
	}

	return false, nil
}

func (r *Reconciler) getOperatorImage() (string, error) {
	var operatorPod v1.Pod
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: r.operatorPodName, Namespace: r.operatorNamespace}, &operatorPod); err != nil {
		return "", errors.WithStack(err)
	}

	if operatorPod.Spec.Containers == nil || len(operatorPod.Spec.Containers) < 1 {
		return "", errors.New("invalid operator pod spec")
	}

	return operatorPod.Spec.Containers[0].Image, nil
}

func buildDesiredCSIDaemonSet(operatorImage, operatorNamespace, saName string) (*appsv1.DaemonSet, error) {
	ds := prepareDaemonSet(operatorImage, operatorNamespace, saName)

	dsHash, err := kubeobjects.GenerateHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[kubeobjects.AnnotationHash] = dsHash

	return ds, nil
}

func prepareDaemonSet(operatorImage, operatorNamespace, saName string) *appsv1.DaemonSet {
	labels := prepareDaemonSetLabels()

	return &appsv1.DaemonSet{
		ObjectMeta: prepareMetadata(operatorNamespace),
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-logs-container": "driver",
					},
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						prepareDriverContainer(operatorImage),
						prepareRegistrarContainer(operatorImage),
						preparelivenessProbeContainer(operatorImage),
					},
					ServiceAccountName: prepareServiceAccount(saName),
					Volumes:            prepareVolumes(),
				},
			},
		},
	}
}

func prepareDaemonSetLabels() map[string]string {
	return map[string]string{
		"internal.oneagent.dynatrace.com/component": "csi-driver",
		"internal.oneagent.dynatrace.com/app":       "csi-driver",
	}
}

func prepareMetadata(namespace string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      DaemonSetName,
		Namespace: namespace,
		Labels: map[string]string{
			"dynatrace.com/operator": "dynatrace",
		},
		Annotations: map[string]string{},
	}
}

func prepareDriverContainer(operatorImage string) v1.Container {
	privileged := true
	userID := int64(0)
	envVars := prepareDriverEnvVars()
	livenessProbe := prepareDriverLivenessProbe()
	volumeMounts := prepareDriverVolumeMounts()

	return v1.Container{
		Name:  "driver",
		Image: operatorImage,
		Args: []string{
			"csi-driver",
			"--endpoint=unix://csi/csi.sock",
			"--node-id=$(KUBE_NODE_NAME)",
			"--health-probe-bind-address=:10080",
		},
		Env:             envVars,
		ImagePullPolicy: v1.PullAlways,
		Ports: []v1.ContainerPort{
			{
				Name:          "healthz",
				Protocol:      v1.ProtocolTCP,
				ContainerPort: 10080,
			},
		},
		LivenessProbe: &livenessProbe,
		SecurityContext: &v1.SecurityContext{
			Privileged: &privileged,
			RunAsUser:  &userID,
			SELinuxOptions: &v1.SELinuxOptions{
				Level: "s0",
			},
		},
		VolumeMounts: volumeMounts,
	}
}

func prepareDriverEnvVars() []v1.EnvVar {
	return []v1.EnvVar{
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "KUBE_NODE_NAME",
			ValueFrom: &v1.EnvVarSource{
				FieldRef: &v1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "spec.nodeName",
				},
			},
		},
	}
}

func prepareDriverLivenessProbe() v1.Probe {
	return v1.Probe{
		FailureThreshold:    3,
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		TimeoutSeconds:      1,
		Handler:             prepareLivenessProbeHandler(),
	}
}

func prepareDriverVolumeMounts() []v1.VolumeMount {
	bidirectional := v1.MountPropagationBidirectional
	return []v1.VolumeMount{
		{
			Name:      "plugin-dir",
			MountPath: "/csi",
		},
		{
			Name:             "plugins-dir",
			MountPath:        "/var/lib/kubelet/plugins",
			MountPropagation: &bidirectional,
		},
		{
			Name:             "mountpoint-dir",
			MountPath:        "/var/lib/kubelet/pods",
			MountPropagation: &bidirectional,
		},
		{
			Name:             "dynatrace-oneagent-data-dir",
			MountPath:        "/data",
			MountPropagation: &bidirectional,
		},
	}
}

func prepareRegistrarContainer(operatorImage string) v1.Container {
	userID := int64(0)
	livenessProbe := prepareRegistrarLivenessProbe()
	volumeMounts := prepareRegistrarVolumeMounts()

	return v1.Container{
		Name:            "registrar",
		Image:           operatorImage,
		ImagePullPolicy: v1.PullAlways,
		Command: []string{
			"csi-node-driver-registrar",
		},
		Args: []string{
			"--csi-address=/csi/csi.sock",
			"--kubelet-registration-path=/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com/csi.sock",
			"--health-port=9809",
		},
		Ports: []v1.ContainerPort{
			{
				Name:          "healthz",
				ContainerPort: 9809,
			},
		},
		LivenessProbe: &livenessProbe,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &userID,
		},
		VolumeMounts: volumeMounts,
	}
}

func prepareRegistrarLivenessProbe() v1.Probe {
	return v1.Probe{
		InitialDelaySeconds: 5,
		TimeoutSeconds:      5,
		Handler:             prepareLivenessProbeHandler(),
	}
}

func prepareLivenessProbeHandler() v1.Handler {
	return v1.Handler{
		HTTPGet: &v1.HTTPGetAction{
			Path: "/healthz",
			Port: intstr.FromString("healthz"),
		},
	}
}

func prepareRegistrarVolumeMounts() []v1.VolumeMount {
	return []v1.VolumeMount{
		{
			Name:      "plugin-dir",
			MountPath: "/csi",
		},
		{
			Name:      "registration-dir",
			MountPath: "/registration",
		},
	}
}

func preparelivenessProbeContainer(operatorImage string) v1.Container {
	return v1.Container{
		Name:            "liveness-probe",
		Image:           operatorImage,
		ImagePullPolicy: v1.PullAlways,
		Command: []string{
			"livenessprobe",
		},
		Args: []string{
			"--csi-address=/csi/csi.sock",
			"--health-port=9898",
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "plugin-dir",
				MountPath: "/csi",
			},
		},
	}
}

func prepareServiceAccount(saName string) string {
	serviceAccountName := DefaultServiceAccountName
	if saName != "" {
		serviceAccountName = saName
	}
	return serviceAccountName
}

func prepareVolumes() []v1.Volume {
	hostPathDir := v1.HostPathDirectory
	hostPathDirOrCreate := v1.HostPathDirectoryOrCreate

	return []v1.Volume{
		{
			Name: "registration-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: RegistrarDirPath,
					Type: &hostPathDir,
				},
			},
		},
		{
			Name: "plugin-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: PluginDirPath,
					Type: &hostPathDirOrCreate,
				},
			},
		},
		{
			Name: "plugins-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: PluginsDirPath,
					Type: &hostPathDir,
				},
			},
		},
		{
			Name: "mountpoint-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: MountpointDirPath,
					Type: &hostPathDirOrCreate,
				},
			},
		},
		{
			Name: "dynatrace-oneagent-data-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: OneAgentDataDirPath,
					Type: &hostPathDirOrCreate,
				},
			},
		},
	}
}
