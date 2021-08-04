package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/Dynatrace/dynatrace-operator/webhook/script"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	missingDynakubeEvent = "MissingDynakube"
)

var (
	logger = log.Log.WithName("oneagent.webhook")
	debug  = os.Getenv("DEBUG_OPERATOR")
)

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
		client:    mgr.GetClient(),
		namespace: ns,
		image:     pod.Spec.Containers[0].Image,
		apmExists: apmExists,
		clusterID: string(UID),
		recorder:  mgr.GetEventRecorderFor("Webhook Server"),
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
	recorder  record.EventRecorder
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

	var dk dynatracev1alpha1.DynaKube
	if err := m.client.Get(ctx, client.ObjectKey{Name: oaName, Namespace: m.namespace}, &dk); k8serrors.IsNotFound(err) {
		template := "namespace '%s' is assigned to DynaKube instance '%s' but doesn't exist"
		m.recorder.Eventf(
			&dynatracev1alpha1.DynaKube{ObjectMeta: v1.ObjectMeta{Name: "placeholder", Namespace: m.namespace}},
			corev1.EventTypeWarning,
			missingDynakubeEvent,
			template, req.Namespace, oaName)
		return admission.Errored(http.StatusBadRequest, fmt.Errorf(
			template, req.Namespace, oaName))
	} else if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if !dk.Spec.CodeModules.Enabled {
		logger.Info("injection disabled")
		return admission.Patched("")
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	if pod.Annotations[dtwebhook.AnnotationInjected] == "true" {
		if dk.FeatureEnableWebhookReinvocationPolicy() {
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
					initGenerator := script.NewInitGenerator(m.client, &dk, &ns, pod)
					init, _ := initGenerator.NewScript(ctx)

					deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID)
					updateContainer(c, &dk, pod, deploymentMetadata, init["proxy"])

					if installContainer == nil {
						for j := range pod.Spec.InitContainers {
							ic := &pod.Spec.InitContainers[j]

							if ic.Name == dtwebhook.InstallContainerName {
								installContainer = ic
								break
							}
						}
					}
					installContainer.Env = []corev1.EnvVar{
						{Name: "INIT", Value: init["init.sh"]},
						{Name: "CA", Value: init["ca.pem"]},
						{Name: "K8S_PODNAME", ValueFrom: fieldEnvVar("metadata.name")},
						{Name: "K8S_PODUID", ValueFrom: fieldEnvVar("metadata.uid")},
						{Name: "K8S_NODE_NAME", ValueFrom: fieldEnvVar("spec.nodeName")},
					}
					needsUpdate = true
				}
			}

			if needsUpdate {
				logger.Info("updating pod with missing containers")
				m.recorder.Eventf(&dk,
					corev1.EventTypeNormal,
					updatePodEvent,
					"Updating pod %s in namespace %s with missing containers", pod.GenerateName, pod.Namespace)
				return getResponse(pod, &req)
			}
		}

		return admission.Patched("")
	}
	pod.Annotations[dtwebhook.AnnotationInjected] = "true"
	image := m.image

	dkVol := dk.Spec.CodeModules.Volume
	if dkVol == (corev1.VolumeSource{}) {
		dkVol.CSI = &corev1.CSIVolumeSource{
			Driver: dtcsi.DriverName,
		}
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{Name: dtwebhook.OneagentBinMount, VolumeSource: dkVol},
		corev1.Volume{
			Name: dtwebhook.OneagentShareMount,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: dtwebhook.OneagentConfigMount,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})

	var sc *corev1.SecurityContext
	if pod.Spec.Containers[0].SecurityContext != nil {
		sc = pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}

	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID)

	initGenerator := script.NewInitGenerator(m.client, &dk, &ns, pod)
	init, err := initGenerator.NewScript(ctx)
	if err != nil {
		logger.Error(err, "something broke")
	}

	ic := corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullAlways,
		Command:         []string{"/usr/bin/env"},
		Args:            []string{"echo", "${INIT}", "|", "base64", "-d", ">", "./init.sh", "bash", "./init.sh"},
		Env: []corev1.EnvVar{
			{Name: "INIT", Value: init["init.sh"]},
			{Name: "CA", Value: init["ca.pem"]},
			{Name: "K8S_PODNAME", ValueFrom: fieldEnvVar("metadata.name")},
			{Name: "K8S_PODUID", ValueFrom: fieldEnvVar("metadata.uid")},
			{Name: "K8S_NODE_NAME", ValueFrom: fieldEnvVar("spec.nodeName")},
		},
		SecurityContext: sc,
		VolumeMounts: []corev1.VolumeMount{
			{Name: dtwebhook.OneagentBinMount, MountPath: dtwebhook.InitBinDir},
			{Name: dtwebhook.OneagentShareMount, MountPath: dtwebhook.InitShareDir},
			{Name: dtwebhook.OneagentConfigMount, MountPath: dtwebhook.InitConfigDir},
		},
		Resources: dk.Spec.CodeModules.Resources,
	}

	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		updateContainer(c, &dk, pod, deploymentMetadata, init["proxy"])
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, ic)

	m.recorder.Eventf(&dk,
		corev1.EventTypeNormal,
		injectEvent,
		"Injecting the necessary info into pod %s in namespace %s", pod.Name, ns.Name)
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

func fieldEnvVar(key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: key}}
}

// updateContainer sets missing preload Variables
func updateContainer(c *corev1.Container, oa *dynatracev1alpha1.DynaKube,
	pod *corev1.Pod, deploymentMetadata *deploymentmetadata.DeploymentMetadata, proxy string) {

	logger.Info("updating container with missing preload variables", "containerName", c.Name)
	installPath := utils.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	c.VolumeMounts = append(c.VolumeMounts,
		corev1.VolumeMount{
			Name:      dtwebhook.OneagentShareMount,
			MountPath: dtwebhook.LDSOPreloadPath,
			SubPath:   "ld.so.preload",
		},
		corev1.VolumeMount{
			Name:      dtwebhook.OneagentBinMount,
			MountPath: installPath,
		},
		corev1.VolumeMount{
			Name:      dtwebhook.OneagentShareMount,
			MountPath: dtwebhook.ConfMountPath,
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

	if proxy != "" {
		c.Env = append(c.Env,
			corev1.EnvVar{
				Name:  "DT_PROXY",
				Value: proxy,
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
