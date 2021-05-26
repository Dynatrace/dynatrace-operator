package dtcsi

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/Dynatrace/dynatrace-operator/controllers/activegate"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ReconcileCSI struct {
	client   client.Client
	scheme   *runtime.Scheme
	logger   logr.Logger
	instance *v1alpha1.DynaKube
}

func NewCSIReconciler(client client.Client, scheme *runtime.Scheme, logger logr.Logger, instance *v1alpha1.DynaKube) *ReconcileCSI {
	return &ReconcileCSI{
		client:   client,
		scheme:   scheme,
		logger:   logger,
		instance: instance,
	}
}

func (r *ReconcileCSI) Reconcile() (bool, error) {
	r.logger.Info("Reconciling CSI driver")

	driver := buildCSIDriver()
	created, err := r.createCSIDriverIfNotExists(driver)
	if created || err != nil {
		return created, errors.WithStack(err)
	}

	ds, err := buildDesiredCSIDaemonSet()
	if err != nil {
		return false, errors.WithStack(err)
	}

	if err := controllerutil.SetControllerReference(r.instance, ds, r.scheme); err != nil {
		return false, errors.WithStack(err)
	}

	created, err = r.createDaemonSetIfNotExists(ds)
	if created || err != nil {
		return created, errors.WithStack(err)
	}

	updated, err := r.updateDaemonSetIfOutdated(ds)
	if updated || err != nil {
		return updated, errors.WithStack(err)
	}

	return false, nil
}

func buildCSIDriver() *v12.CSIDriver {
	trueVal := true
	falseVal := false
	return &v12.CSIDriver{
		ObjectMeta: metav1.ObjectMeta{
			Name: DriverName,
		},
		Spec: v12.CSIDriverSpec{
			AttachRequired: &falseVal,
			PodInfoOnMount: &trueVal,
			VolumeLifecycleModes: []v12.VolumeLifecycleMode{
				v12.VolumeLifecycleEphemeral,
			},
		},
	}
}

func (r *ReconcileCSI) createCSIDriverIfNotExists(desiredCSIDriver *v12.CSIDriver) (bool, error) {
	_, err := r.getCSIDriver(desiredCSIDriver)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		r.logger.Info("creating new CSI driver")
		return true, r.client.Create(context.TODO(), desiredCSIDriver)
	}
	return false, err
}

func (r *ReconcileCSI) getCSIDriver(desiredCSIDriver *v12.CSIDriver) (*v12.CSIDriver, error) {
	var actualDriver v12.CSIDriver
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: desiredCSIDriver.Name, Namespace: desiredCSIDriver.Namespace}, &actualDriver)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &actualDriver, nil
}

func (r *ReconcileCSI) createDaemonSetIfNotExists(desiredSts *appsv1.DaemonSet) (bool, error) {
	_, err := r.getDaemonSet(desiredSts)
	if err != nil && k8serrors.IsNotFound(errors.Cause(err)) {
		r.logger.Info("creating new daemonset set for CSI driver")
		return true, r.client.Create(context.TODO(), desiredSts)
	}
	return false, err
}

func (r *ReconcileCSI) getDaemonSet(desiredDs *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	var actualDs appsv1.DaemonSet
	err := r.client.Get(context.TODO(), client.ObjectKey{Name: desiredDs.Name, Namespace: desiredDs.Namespace}, &actualDs)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &actualDs, nil
}

func (r *ReconcileCSI) updateDaemonSetIfOutdated(desiredDs *appsv1.DaemonSet) (bool, error) {
	currentSts, err := r.getDaemonSet(desiredDs)
	if err != nil {
		return false, err
	}
	if !hasDaemonSetChanged(currentSts, desiredDs) {
		return false, nil
	}

	r.logger.Info("updating existing CSI driver daemonset")
	if err = r.client.Update(context.TODO(), desiredDs); err != nil {
		return false, err
	}
	return true, err
}

func hasDaemonSetChanged(a, b *appsv1.DaemonSet) bool {
	return getTemplateHash(a) != getTemplateHash(b)
}

func getTemplateHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[activegate.AnnotationTemplateHash]
	}
	return ""
}

func buildDesiredCSIDaemonSet() (*appsv1.DaemonSet, error) {
	metadata := prepareMetadata()
	driver := prepareDriverSpec()
	registrar := prepareRegistrarSpec()
	livenessprobe := preparelivenessProbeSpec()
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
	}
}

func prepareDriverSpec() v1.Container {
	privileged := true
	userID := int64(0)
	bidirectional := v1.MountPropagationBidirectional

	return v1.Container{
		Name:    "driver",
		Image:   "quay.io/dynatrace/dynatrace-operator:snapshot",
		Command: []string{"csi-driver"},
		Args: []string{
			"--endpoint=unix://csi/csi.sock",
			"--node-id=$(KUBE_NODE_NAME)",
			"--health-probe-bind-address=:10080",
		},
		Env: []v1.EnvVar{
			{
				Name:  "GC_INTERVAL_MINUTES",
				Value: "60",
			},
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

func prepareRegistrarSpec() v1.Container {
	userID := int64(0)

	return v1.Container{
		Name:            "registrar",
		Image:           "quay.io/dynatrace/dynatrace-operator:snapshot",
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
		},
	}
}

func preparelivenessProbeSpec() v1.Container {
	return v1.Container{
		Name:  "liveness-probe",
		Image: "quay.io/dynatrace/dynatrace-operator:snapshot",
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
