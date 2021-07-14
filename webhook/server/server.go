package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var logger = log.Log.WithName("oneagent.webhook")
var debug = os.Getenv("DEBUG_OPERATOR")

// AddToManager adds the Webhook server to the Manager
func AddToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		logger.Info("No Pod name set for webhook container")
	}

	if podName == "" && debug == "true" {
		registerDebugInjectEndpoint(mgr, ns)
	} else {
		if err := registerInjectEndpoint(mgr, ns, podName); err != nil {
			return err
		}
	}

	registerHealthzEndpoint(mgr)
	return nil
}

// registerDebugInjectEndpoint registers an endpoint at /inject with an empty image
//
// If the webhook runs in a non-debug environment, the webhook should exit if no
// pod with a given POD_NAME is found. It needs this pod to set the image for the podInjector
// When debugging, the Webhook should not exit in this scenario, but register the endpoint with an empty image
// to allow further debugging steps.
//
// This behavior must only occur if the DEBUG_OPERATOR flag is set to true
func registerDebugInjectEndpoint(mgr manager.Manager, ns string) {
	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podInjector{
		namespace: ns,
	}})
}

func registerInjectEndpoint(mgr manager.Manager, ns string, podName string) error {
	// Don't use mgr.GetClient() on this function, or other cache-dependent functions from the manager. The cache may
	// not be ready at this point, and queries for Kubernetes objects may fail. mgr.GetAPIReader() doesn't depend on the
	// cache and is safe to use.

	apmExists, err := utils.CheckIfOneAgentAPMExists(mgr.GetConfig())
	if err != nil {
		return err
	}
	if apmExists {
		logger.Info("OneAgentAPM object detected - DynaKube webhook won't inject until the OneAgent Operator has been uninstalled")
	}

	var pod corev1.Pod
	if err := mgr.GetAPIReader().Get(context.TODO(), client.ObjectKey{
		Name:      podName,
		Namespace: ns,
	}, &pod); err != nil {
		return err
	}

	var UID types.UID
	if UID, err = kubesystem.GetUID(mgr.GetAPIReader()); err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podInjector{
		namespace: ns,
		image:     pod.Spec.Containers[0].Image,
		apmExists: apmExists,
		clusterID: string(UID),
	}})
	return nil
}

func registerHealthzEndpoint(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

// podAnnotator injects the OneAgent into Pods
type podInjector struct {
	client    client.Client
	decoder   *admission.Decoder
	image     string
	namespace string
	apmExists bool
	clusterID string
}

// podAnnotator adds an annotation to every incoming pods
func (m *podInjector) Handle(ctx context.Context, req admission.Request) admission.Response {
	if m.apmExists {
		return admission.Patched("")
	}

	pod := &corev1.Pod{}

	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	logger.Info("injecting into Pod", "name", pod.Name, "generatedName", pod.GenerateName, "namespace", req.Namespace)

	var ns corev1.Namespace
	if err := m.client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &ns); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	inject := utils.GetField(ns.Annotations, dtwebhook.AnnotationInject, "true")
	inject = utils.GetField(pod.Annotations, dtwebhook.AnnotationInject, inject)
	if inject == "false" {
		return admission.Patched("")
	}

	oaName := utils.GetField(ns.Labels, dtwebhook.LabelInstance, "")
	if oaName == "" {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("no DynaKube instance set for namespace: %s", req.Namespace))
	}

	var oa dynatracev1alpha1.DynaKube
	if err := m.client.Get(ctx, client.ObjectKey{Name: oaName, Namespace: m.namespace}, &oa); k8serrors.IsNotFound(err) {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf(
			"namespace '%s' is assigned to DynaKube instance '%s' but doesn't exist", req.Namespace, oaName))
	} else if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if !oa.Spec.CodeModules.Enabled {
		logger.Info("injection disabled")
		return admission.Patched("")
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	if pod.Annotations[dtwebhook.AnnotationInjected] == "true" {
		if oa.FeatureEnableWebhookReinvocationPolicy() {
			var needsUpdate = false
			var installContainer *corev1.Container
			for i := range pod.Spec.Containers {
				c := &pod.Spec.Containers[i]

				preloaded := false
				for _, e := range c.Env {
					if e.Name == "LD_PRELOAD" {
						preloaded = true
						break
					}
				}

				if !preloaded {
					// container does not have LD_PRELOAD set
					logger.Info("instrumenting missing container", "name", c.Name)

					deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID, oa)
					updateContainer(c, &oa, pod, deploymentMetadata)

					if installContainer == nil {
						for j := range pod.Spec.InitContainers {
							ic := &pod.Spec.InitContainers[j]

							if ic.Name == dtwebhook.InstallContainerName {
								installContainer = ic
								break
							}
						}
					}
					updateInstallContainer(installContainer, i+1, c.Name, c.Image)

					needsUpdate = true
				}
			}

			if needsUpdate {
				logger.Info("updating pod with missing containers")
				return getResponse(pod, &req)
			}
		}

		return admission.Patched("")
	}
	pod.Annotations[dtwebhook.AnnotationInjected] = "true"

	technologies := url.QueryEscape(utils.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all"))
	installPath := utils.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)
	installerURL := utils.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, "")
	failurePolicy := utils.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")
	image := m.image

	dkVol := oa.Spec.CodeModules.Volume
	if dkVol == (corev1.VolumeSource{}) {
		dkVol.CSI = &corev1.CSIVolumeSource{
			Driver: dtcsi.DriverName,
		}
	}

	mode := "provisioned"
	if dkVol.EmptyDir != nil {
		mode = "installer"
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{Name: "oneagent-bin", VolumeSource: dkVol},
		corev1.Volume{
			Name: "oneagent-share",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: "oneagent-config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dtwebhook.SecretConfigName,
				},
			},
		})

	var sc *corev1.SecurityContext
	if pod.Spec.Containers[0].SecurityContext != nil {
		sc = pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}

	fieldEnvVar := func(key string) *corev1.EnvVarSource {
		return &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: key}}
	}

	basePodName := pod.GenerateName
	if basePodName == "" {
		basePodName = pod.Name
	}

	// Only include up to the last dash character, exclusive.
	if p := strings.LastIndex(basePodName, "-"); p != -1 {
		basePodName = basePodName[:p]
	}

	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID, oa)

	ic := corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/usr/bin/env"},
		Args:            []string{"bash", "/mnt/config/init.sh"},
		Env: []corev1.EnvVar{
			{Name: "FLAVOR", Value: dtclient.FlavorMultidistro},
			{Name: "TECHNOLOGIES", Value: technologies},
			{Name: "INSTALLPATH", Value: installPath},
			{Name: "INSTALLER_URL", Value: installerURL},
			{Name: "FAILURE_POLICY", Value: failurePolicy},
			{Name: "CONTAINERS_COUNT", Value: strconv.Itoa(len(pod.Spec.Containers))},
			{Name: "MODE", Value: mode},
			{Name: "K8S_PODNAME", ValueFrom: fieldEnvVar("metadata.name")},
			{Name: "K8S_PODUID", ValueFrom: fieldEnvVar("metadata.uid")},
			{Name: "K8S_BASEPODNAME", Value: basePodName},
			{Name: "K8S_NAMESPACE", ValueFrom: fieldEnvVar("metadata.namespace")},
			{Name: "K8S_NODE_NAME", ValueFrom: fieldEnvVar("spec.nodeName")},
		},
		SecurityContext: sc,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "oneagent-bin", MountPath: "/mnt/bin"},
			{Name: "oneagent-share", MountPath: "/mnt/share"},
			{Name: "oneagent-config", MountPath: "/mnt/config"},
		},
		Resources: oa.Spec.CodeModules.Resources,
	}

	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]

		updateInstallContainer(&ic, i+1, c.Name, c.Image)

		updateContainer(c, &oa, pod, deploymentMetadata)
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, ic)

	return getResponse(pod, &req)
}

// InjectClient injects the client
func (m *podInjector) InjectClient(c client.Client) error {
	m.client = c
	return nil
}

// InjectDecoder injects the decoder
func (m *podInjector) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

// updateInstallContainer adds Container to list of Containers of Install Container
func updateInstallContainer(ic *corev1.Container, number int, name string, image string) {
	logger.Info("updating install container with new container", "containerName", name, "containerImage", image)
	ic.Env = append(ic.Env,
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_NAME", number), Value: name},
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_IMAGE", number), Value: image})
}

// updateContainer sets missing preload Variables
func updateContainer(c *corev1.Container, oa *dynatracev1alpha1.DynaKube,
	pod *corev1.Pod, deploymentMetadata *deploymentmetadata.DeploymentMetadata) {

	logger.Info("updating container with missing preload variables", "containerName", c.Name)
	installPath := utils.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	c.VolumeMounts = append(c.VolumeMounts,
		corev1.VolumeMount{
			Name:      "oneagent-share",
			MountPath: "/etc/ld.so.preload",
			SubPath:   "ld.so.preload",
		},
		corev1.VolumeMount{
			Name:      "oneagent-bin",
			MountPath: installPath,
		},
		corev1.VolumeMount{
			Name:      "oneagent-share",
			MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
			SubPath:   fmt.Sprintf("container_%s.conf", c.Name),
		})

	c.Env = append(c.Env,
		corev1.EnvVar{
			Name:  "LD_PRELOAD",
			Value: installPath + "/agent/lib64/liboneagentproc.so",
		},
		corev1.EnvVar{
			Name:  "DT_DEPLOYMENT_METADATA",
			Value: deploymentMetadata.AsString(),
		})

	if oa.Spec.Proxy != nil && (oa.Spec.Proxy.Value != "" || oa.Spec.Proxy.ValueFrom != "") {
		c.Env = append(c.Env,
			corev1.EnvVar{
				Name: "DT_PROXY",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: dtwebhook.SecretConfigName,
						},
						Key: "proxy",
					},
				},
			})
	}

	if oa.Spec.NetworkZone != "" {
		c.Env = append(c.Env, corev1.EnvVar{Name: "DT_NETWORK_ZONE", Value: oa.Spec.NetworkZone})
	}
}

// getResponse tries to format pod as json
func getResponse(pod *corev1.Pod, req *admission.Request) admission.Response {
	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
