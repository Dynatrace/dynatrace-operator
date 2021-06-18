package dtcsi

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"os"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
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

type Reconciler struct {
	client   client.Client
	scheme   *runtime.Scheme
	logger   logr.Logger
	instance *v1alpha1.DynaKube
}

func NewReconciler(client client.Client, scheme *runtime.Scheme, logger logr.Logger, instance *v1alpha1.DynaKube) *Reconciler {
	return &Reconciler{
		client:   client,
		scheme:   scheme,
		logger:   logger,
		instance: instance,
	}
}

func (r *Reconciler) Reconcile() (bool, error) {
	r.logger.Info("Reconciling CSI driver")

	operatorImage, err := r.getOperatorImage()
	if err != nil {
		return false, errors.WithStack(err)
	}

	ds, err := buildDesiredCSIDaemonSet(operatorImage)
	if err != nil {
		return false, errors.WithStack(err)
	}

	if err := controllerutil.SetControllerReference(r.instance, ds, r.scheme); err != nil {
		return false, errors.WithStack(err)
	}

	upd, err := utils.CreateOrUpdateDaemonSet(r.client, r.logger, ds)
	if upd || err != nil {
		return upd, errors.WithStack(err)
	}

	return false, nil
}

func (r *Reconciler) getOperatorImage() (string, error) {
	var operatorPod v1.Pod
	operatorPodName := os.Getenv("POD_NAME")
	operatorNamespace := os.Getenv("POD_NAMESPACE")

	if err := r.client.Get(context.TODO(), client.ObjectKey{Name: operatorPodName, Namespace: operatorNamespace}, &operatorPod); err != nil {
		return "", errors.WithStack(err)
	}

	return operatorPod.Spec.Containers[0].Image, nil
}

func buildDesiredCSIDaemonSet(operatorImage string) (*appsv1.DaemonSet, error) {
	metadata := prepareMetadata()
	driver := prepareDriverSpec(operatorImage)
	registrar := prepareRegistrarSpec(operatorImage)
	livenessprobe := preparelivenessProbeSpec(operatorImage)
	vol := prepareVolumes()

	ds := &appsv1.DaemonSet{
		ObjectMeta: metadata,
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"internal.oneagent.dynatrace.com/component": "csi-driver",
					"internal.oneagent.dynatrace.com/app":       "csi-driver",
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"kubectl.kubernetes.io/default-logs-container": "driver",
					},
					Labels: map[string]string{
						"internal.oneagent.dynatrace.com/component": "csi-driver",
						"internal.oneagent.dynatrace.com/app":       "csi-driver",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						driver,
						registrar,
						livenessprobe,
					},
					ServiceAccountName: DaemonSetName,
					Volumes:            vol,
				},
			},
		},
	}

	dsHash, err := generateDaemonSetHash(ds)
	if err != nil {
		return nil, err
	}
	ds.Annotations[activegate.AnnotationTemplateHash] = dsHash

	return ds, nil
}

func generateDaemonSetHash(ds *appsv1.DaemonSet) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func prepareMetadata() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      DaemonSetName,
		Namespace: "dynatrace",
		Labels: map[string]string{
			"dynatrace.com/operator": "dynatrace",
		},
		Annotations: map[string]string{},
	}
}

func prepareDriverSpec(operatorImage string) v1.Container {
	privileged := true
	userID := int64(0)
	bidirectional := v1.MountPropagationBidirectional

	return v1.Container{
		Name:    "driver",
		Image:   operatorImage,
		Command: []string{"csi-driver"},
		Args: []string{
			"--endpoint=unix://csi/csi.sock",
			"--node-id=$(KUBE_NODE_NAME)",
			"--health-probe-bind-address=:10080",
		},
		Env: []v1.EnvVar{
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
		},
		ImagePullPolicy: v1.PullAlways,
		Ports: []v1.ContainerPort{
			{
				Name:          "healthz",
				Protocol:      v1.ProtocolTCP,
				ContainerPort: 10080,
			},
		},
		LivenessProbe: &v1.Probe{
			FailureThreshold:    3,
			InitialDelaySeconds: 5,
			PeriodSeconds:       5,
			SuccessThreshold:    1,
			TimeoutSeconds:      1,
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   "/healthz",
					Port:   intstr.FromString("healthz"),
					Scheme: "HTTP",
				},
			},
		},
		ReadinessProbe: &v1.Probe{
			FailureThreshold:    3,
			InitialDelaySeconds: 5,
			PeriodSeconds:       5,
			SuccessThreshold:    1,
			TimeoutSeconds:      1,
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path:   "/readyz",
					Port:   intstr.FromString("healthz"),
					Scheme: "HTTP",
				},
			},
		},
		SecurityContext: &v1.SecurityContext{
			Privileged: &privileged,
			RunAsUser:  &userID,
			SELinuxOptions: &v1.SELinuxOptions{
				Level: "s0",
			},
		},
		VolumeMounts: []v1.VolumeMount{
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
		},
	}
}

func prepareRegistrarSpec(operatorImage string) v1.Container {
	userID := int64(0)

	return v1.Container{
		Name:            "registrar",
		Image:           operatorImage,
		ImagePullPolicy: v1.PullIfNotPresent,
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
		LivenessProbe: &v1.Probe{
			InitialDelaySeconds: 5,
			TimeoutSeconds:      5,
			Handler: v1.Handler{
				HTTPGet: &v1.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromString("healthz"),
				},
			},
		},
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &userID,
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "plugin-dir",
				MountPath: "/csi",
			},
			{
				Name:      "registration-dir",
				MountPath: "/registration",
			},
		},
	}
}

func preparelivenessProbeSpec(operatorImage string) v1.Container {
	return v1.Container{
		Name:  "liveness-probe",
		Image: operatorImage,
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

func prepareVolumes() []v1.Volume {
	hostPathDir := v1.HostPathDirectory
	hostPathDirOrCreate := v1.HostPathDirectoryOrCreate

	return []v1.Volume{
		{
			Name: "registration-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/var/lib/kubelet/plugins_registry/",
					Type: &hostPathDir,
				},
			},
		},
		{
			Name: "plugin-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com",
					Type: &hostPathDirOrCreate,
				},
			},
		},
		{
			Name: "plugins-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/var/lib/kubelet/plugins",
					Type: &hostPathDir,
				},
			},
		},
		{
			Name: "mountpoint-dir",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/var/lib/kubelet/pods",
					Type: &hostPathDirOrCreate,
				},
			},
		},
	}
}
