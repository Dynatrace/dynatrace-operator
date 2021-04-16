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
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/controllers/utils"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
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

	logger.Info("checking pod", "name", pod.Name, "generatedName", pod.GenerateName, "namespace", req.Namespace)

	codeModules, err := FindCodeModules(ctx, m.client)
	if err != nil {
		logger.Error(err, "error when trying to find DynaKubes with CodeModules enabled")

		// If CodeModules is not enabled, or cannot be found, cannot inject
		return admission.Patched("")
	}
	if len(codeModules) <= 0 {
		logger.Info("could not find any DynaKubes with CodeModules enabled")
		// If CodeModules is not enabled, cannot inject
		return admission.Patched("")
		//return admission.Errored(http.StatusBadRequest, errors.New("no DynaKube instance exists with CodeModules enabled"))
	}

	oa, err := MatchCodeModules(codeModules, pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if oa == nil {
		// If no DynaKube matches the pods labels, do not inject
		return admission.Patched("")
	}

	logger.Info("injecting into pod", "name", pod.Name, "generatedName", pod.GenerateName, "namespace", req.Namespace)

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	if pod.Annotations[dtwebhook.AnnotationInjected] == "true" {
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
		dkVol.EmptyDir = &corev1.EmptyDirVolumeSource{}
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

	deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID)

	ic := corev1.Container{
		Name:            "install-oneagent",
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

		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_NAME", i+1), Value: c.Name},
			corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_IMAGE", i+1), Value: c.Image})

		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name:      "oneagent-share",
				MountPath: "/etc/ld.so.preload",
				SubPath:   "ld.so.preload",
			},
			corev1.VolumeMount{Name: "oneagent-bin", MountPath: installPath},
			corev1.VolumeMount{
				Name:      "oneagent-share",
				MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
				SubPath:   fmt.Sprintf("container_%s.conf", c.Name),
			})

		c.Env = append(c.Env,
			corev1.EnvVar{Name: "LD_PRELOAD", Value: installPath + "/agent/lib64/liboneagentproc.so"},
			corev1.EnvVar{Name: "DT_DEPLOYMENT_METADATA", Value: deploymentMetadata.AsString()})

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

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, ic)

	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
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

func FindCodeModules(ctx context.Context, clt client.Client) ([]dynatracev1alpha1.DynaKube, error) {
	dynaKubeList := &dynatracev1alpha1.DynaKubeList{}
	err := clt.List(ctx, dynaKubeList)
	if err != nil {
		return nil, errors.Cause(err)
	}

	var codeModules []dynatracev1alpha1.DynaKube
	for _, pod := range dynaKubeList.Items {
		if pod.Spec.CodeModules.Enabled {
			codeModules = append(codeModules, pod)
		}
	}

	return codeModules, nil
}

func MatchCodeModules(codeModules []dynatracev1alpha1.DynaKube, pod *corev1.Pod) (*dynatracev1alpha1.DynaKube, error) {
	var matchingModules []dynatracev1alpha1.DynaKube

	for _, codeModule := range codeModules {
		if areLabelsMatching(codeModule.Spec.CodeModules.Selector.MatchLabels, pod.Labels) {
			expressionsMatching, err := areExpressionsMatching(codeModule.Spec.CodeModules.Selector.MatchExpressions, pod.Labels)
			if err != nil {
				return nil, err
			}
			if expressionsMatching {
				matchingModules = append(matchingModules, codeModule)
			}
		}
	}

	if len(matchingModules) > 1 {
		return nil, errors.New("pod matches two DynaKubes which is unsupported. " +
			"refine the labels on your pod metadata or DynaKube/CodeModules specification")
	}
	if len(matchingModules) == 0 {
		return nil, nil
	}
	return &matchingModules[0], nil
}

func areExpressionsMatching(expressions []metav1.LabelSelectorRequirement, podLabels map[string]string) (bool, error) {
	selector := labels.NewSelector()
	for _, expression := range expressions {
		requirement, err := labels.NewRequirement(expression.Key, selection.Operator(strings.ToLower(string(expression.Operator))), expression.Values)
		if err != nil {
			return false, err
		}
		selector = selector.Add(*requirement)
	}
	return selector.Matches(labels.Set(podLabels)), nil
}

func areLabelsMatching(matchLabels map[string]string, labels map[string]string) bool {
	if len(labels) == 0 {
		return false
	}

	for matchLabel, matchValue := range matchLabels {
		value, ok := labels[matchLabel]
		if !ok || matchValue != value {
			return false
		}
	}
	return true
}
