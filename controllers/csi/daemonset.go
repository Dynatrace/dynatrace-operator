package dtcsi

import (
	"context"
	"encoding/json"
	"fmt"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	RegistrarDirPath    = "/var/lib/kubelet/plugins_registry/"
	PluginDirPath       = "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com"
	PluginsDirPath      = "/var/lib/kubelet/plugins"
	MountpointDirPath   = "/var/lib/kubelet/pods"
	OneAgentDataDirPath = "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com/data"

	driverDefaultCPU    = 300
	driverDefaultMemory = 100

	registrarDefaultCPU    = 10
	registrarDefaultMemory = 15

	livenessProbeDefaultCPU    = 5
	livenessProbeDefaultMemory = 15
)

type Reconciler struct {
	client            client.Client
	scheme            *runtime.Scheme
	logger            logr.Logger
	instance          *dynatracev1beta1.DynaKube
	operatorPodName   string
	operatorNamespace string
}

func NewReconciler(
	client client.Client, scheme *runtime.Scheme, logger logr.Logger,
	instance *dynatracev1beta1.DynaKube, operatorPodName, operatorNamespace string) *Reconciler {
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

	driverContainerResources := prepareDriverResources(r.client, r.operatorNamespace, r.logger)

	ds, err := buildDesiredCSIDaemonSet(
		operatorImage, r.operatorNamespace, r.instance, driverContainerResources)
	if err != nil {
		return false, errors.WithStack(err)
	}

	upd, err := kubeobjects.CreateOrUpdateDaemonSet(r.client, r.logger, ds)
	if upd || err != nil {
		return upd, errors.WithStack(err)
	}

	return false, nil
}

func (r *Reconciler) getOperatorImage() (string, error) {
	var operatorPod corev1.Pod
	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: r.operatorPodName, Namespace: r.operatorNamespace}, &operatorPod); err != nil {
		return "", errors.WithStack(err)
	}

	if operatorPod.Spec.Containers == nil || len(operatorPod.Spec.Containers) < 1 {
		return "", errors.New("invalid operator pod spec")
	}

	return operatorPod.Spec.Containers[0].Image, nil
}

func buildDesiredCSIDaemonSet(operatorImage, operatorNamespace string, dynakube *dynatracev1beta1.DynaKube,
	driverContainerResources corev1.ResourceRequirements) (*appsv1.DaemonSet, error) {
	ds := prepareDaemonSet(operatorImage, operatorNamespace, dynakube, driverContainerResources)

	dsHash, err := kubeobjects.GenerateHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[kubeobjects.AnnotationHash] = dsHash

	return ds, nil
}

func prepareDaemonSet(operatorImage, operatorNamespace string, dynakube *dynatracev1beta1.DynaKube,
	driverContainerResources corev1.ResourceRequirements) *appsv1.DaemonSet {
	labels := prepareDaemonSetLabels()

	return &appsv1.DaemonSet{
		ObjectMeta: prepareMetadata(operatorNamespace, dynakube),
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-logs-container": "driver",
					},
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						prepareDriverContainer(operatorImage, driverContainerResources),
						prepareRegistrarContainer(operatorImage),
						prepareLivenessProbeContainer(operatorImage),
					},
					ServiceAccountName: DefaultServiceAccountName,
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

func prepareMetadata(namespace string, dynakube *dynatracev1beta1.DynaKube) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      DaemonSetName,
		Namespace: namespace,
		Labels: map[string]string{
			"dynatrace.com/operator": "dynatrace",
		},
		Annotations: map[string]string{},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion:         dynakube.APIVersion,
				Kind:               dynakube.Kind,
				Name:               dynakube.Name,
				UID:                dynakube.UID,
				Controller:         pointer.BoolPtr(false),
				BlockOwnerDeletion: pointer.BoolPtr(false),
			},
		},
	}
}

func prepareDriverContainer(operatorImage string, resources corev1.ResourceRequirements) corev1.Container {
	return corev1.Container{
		Name:  "driver",
		Image: operatorImage,
		Args: []string{
			"csi-driver",
			"--endpoint=unix://csi/csi.sock",
			"--node-id=$(KUBE_NODE_NAME)",
			"--health-probe-bind-address=:10080",
		},
		Env:             prepareDriverEnvVars(),
		ImagePullPolicy: corev1.PullAlways,
		Ports: []corev1.ContainerPort{
			{
				Name:          "healthz",
				Protocol:      corev1.ProtocolTCP,
				ContainerPort: 10080,
			},
		},
		Resources:       resources,
		LivenessProbe:   prepareDriverLivenessProbe(),
		SecurityContext: prepareSecurityContext(),
		VolumeMounts:    prepareDriverVolumeMounts(),
	}
}

func prepareDriverEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "KUBE_NODE_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "spec.nodeName",
				},
			},
		},
	}
}

func prepareDriverResources(client client.Client, operatorNS string, logger logr.Logger) corev1.ResourceRequirements {
	deployment, err := kubeobjects.GetDeployment(client, operatorNS)
	if err != nil {
		logger.Info(fmt.Sprintf("failed to get deployment for reading '%s' label", AnnotationCSIResourcesIdentifier), "err", err)
	} else {
		var res corev1.ResourceRequirements

		if label, ok := deployment.Annotations[AnnotationCSIResourcesIdentifier]; ok {
			if err = json.Unmarshal([]byte(label), &res); err != nil {
				logger.Info(fmt.Sprintf("failed to unmarshal '%s' label json", AnnotationCSIResourcesIdentifier), "err", err)
			} else {
				return res
			}
		}
	}

	return prepareResources(driverDefaultCPU, driverDefaultMemory)
}

func getQuantity(value int64, scale resource.Scale) resource.Quantity {
	return *resource.NewScaledQuantity(value, scale)
}

func prepareSecurityContext() *corev1.SecurityContext {
	privileged := true
	userID := int64(0)

	return &corev1.SecurityContext{
		Privileged: &privileged,
		RunAsUser:  &userID,
		SELinuxOptions: &corev1.SELinuxOptions{
			Level: "s0",
		},
	}
}

func prepareDriverLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		FailureThreshold:    3,
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		TimeoutSeconds:      1,
		Handler:             prepareLivenessProbeHandler(),
	}
}

func prepareDriverVolumeMounts() []corev1.VolumeMount {
	bidirectional := corev1.MountPropagationBidirectional
	return []corev1.VolumeMount{
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

func prepareRegistrarContainer(operatorImage string) corev1.Container {
	userID := int64(0)
	livenessProbe := prepareRegistrarLivenessProbe()
	volumeMounts := prepareRegistrarVolumeMounts()

	return corev1.Container{
		Name:            "registrar",
		Image:           operatorImage,
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"csi-node-driver-registrar",
		},
		Args: []string{
			"--csi-address=/csi/csi.sock",
			"--kubelet-registration-path=/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com/csi.sock",
			"--health-port=9809",
		},
		Ports: []corev1.ContainerPort{
			{
				Name:          "healthz",
				ContainerPort: 9809,
			},
		},
		Resources:     prepareResources(registrarDefaultCPU, registrarDefaultMemory),
		LivenessProbe: &livenessProbe,
		SecurityContext: &corev1.SecurityContext{
			RunAsUser: &userID,
		},
		VolumeMounts: volumeMounts,
	}
}

func prepareRegistrarLivenessProbe() corev1.Probe {
	return corev1.Probe{
		InitialDelaySeconds: 5,
		TimeoutSeconds:      5,
		Handler:             prepareLivenessProbeHandler(),
	}
}

func prepareLivenessProbeHandler() corev1.Handler {
	return corev1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Path: "/healthz",
			Port: intstr.FromString("healthz"),
		},
	}
}

func prepareRegistrarVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
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

func prepareLivenessProbeContainer(operatorImage string) corev1.Container {
	return corev1.Container{
		Name:            "liveness-probe",
		Image:           operatorImage,
		ImagePullPolicy: corev1.PullAlways,
		Command: []string{
			"livenessprobe",
		},
		Args: []string{
			"--csi-address=/csi/csi.sock",
			"--health-port=9898",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "plugin-dir",
				MountPath: "/csi",
			},
		},
		Resources: prepareResources(livenessProbeDefaultCPU, livenessProbeDefaultMemory),
	}
}

func prepareVolumes() []corev1.Volume {
	hostPathDir := corev1.HostPathDirectory
	hostPathDirOrCreate := corev1.HostPathDirectoryOrCreate

	return []corev1.Volume{
		{
			Name: "registration-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: RegistrarDirPath,
					Type: &hostPathDir,
				},
			},
		},
		{
			Name: "plugin-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: PluginDirPath,
					Type: &hostPathDirOrCreate,
				},
			},
		},
		{
			Name: "plugins-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: PluginsDirPath,
					Type: &hostPathDir,
				},
			},
		},
		{
			Name: "mountpoint-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: MountpointDirPath,
					Type: &hostPathDirOrCreate,
				},
			},
		},
		{
			Name: "dynatrace-oneagent-data-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: OneAgentDataDirPath,
					Type: &hostPathDirOrCreate,
				},
			},
		},
	}
}

func prepareResources(cpu int64, memory int64) corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    getQuantity(cpu, resource.Milli),
			corev1.ResourceMemory: getQuantity(memory, resource.Mega),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    getQuantity(cpu, resource.Milli),
			corev1.ResourceMemory: getQuantity(memory, resource.Mega),
		},
	}
}
