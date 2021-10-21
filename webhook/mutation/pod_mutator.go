package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/api/v1beta1"
	dtcsi "github.com/Dynatrace/dynatrace-operator/controllers/csi"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/controllers/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/controllers/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/deploymentmetadata"
	"github.com/Dynatrace/dynatrace-operator/dtclient"
	"github.com/Dynatrace/dynatrace-operator/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/logger"
	"github.com/Dynatrace/dynatrace-operator/mapper"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/webhook"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	injectEvent          = "Inject"
	updatePodEvent       = "UpdatePod"
	missingDynakubeEvent = "MissingDynakube"
)

var log = logger.NewDTLogger()
var debug = os.Getenv("DEBUG_OPERATOR")

// AddPodMutationWebhookToManager adds the Webhook server to the Manager
func AddPodMutationWebhookToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		log.Info("No Pod name set for webhook container")
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
	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podMutator{
		namespace: ns,
	}})
}

func registerInjectEndpoint(mgr manager.Manager, ns string, podName string) error {
	// Don't use mgr.GetClient() on this function, or other cache-dependent functions from the manager. The cache may
	// not be ready at this point, and queries for Kubernetes objects may fail. mgr.GetAPIReader() doesn't depend on the
	// cache and is safe to use.

	apmExists, err := kubeobjects.CheckIfOneAgentAPMExists(mgr.GetConfig())
	if err != nil {
		return err
	}
	if apmExists {
		log.Info("OneAgentAPM object detected - DynaKube webhook won't inject until the OneAgent Operator has been uninstalled")
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

	// the injected podMutator.client doesn't have permissions to Get(sth) from a different namespace
	metaClient, err := client.New(mgr.GetConfig(), client.Options{})
	if err != nil {
		return err
	}

	mgr.GetWebhookServer().Register("/inject", &webhook.Admission{Handler: &podMutator{
		metaClient: metaClient,
		apiReader:  mgr.GetAPIReader(),
		namespace:  ns,
		image:      pod.Spec.Containers[0].Image,
		apmExists:  apmExists,
		clusterID:  string(UID),
		recorder:   mgr.GetEventRecorderFor("Webhook Server"),
	}})
	return nil
}

func registerHealthzEndpoint(mgr manager.Manager) {
	mgr.GetWebhookServer().Register("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

// podMutator injects the OneAgent into Pods
type podMutator struct {
	client     client.Client
	metaClient client.Client
	apiReader  client.Reader
	decoder    *admission.Decoder
	image      string
	namespace  string
	apmExists  bool
	clusterID  string
	recorder   record.EventRecorder
}

func rootOwnerPod(ctx context.Context, cnt client.Client, pod *corev1.Pod, namespace string) (string, string, error) {
	obj := &v1.PartialObjectMetadata{
		TypeMeta: v1.TypeMeta{
			APIVersion: pod.APIVersion,
			Kind:       pod.Kind,
		},
		ObjectMeta: v1.ObjectMeta{
			Name: pod.ObjectMeta.Name,
			// pod.ObjectMeta.Namespace is empty yet
			Namespace:       namespace,
			OwnerReferences: pod.ObjectMeta.OwnerReferences,
		},
	}
	return rootOwner(ctx, cnt, obj)
}

func rootOwner(ctx context.Context, cnt client.Client, o *v1.PartialObjectMetadata) (string, string, error) {
	if len(o.ObjectMeta.OwnerReferences) == 0 {
		return o.ObjectMeta.Name, o.Kind, nil
	}

	om := o.ObjectMeta
	for _, owner := range om.OwnerReferences {
		if owner.Controller != nil && *owner.Controller {
			obj := &v1.PartialObjectMetadata{
				TypeMeta: v1.TypeMeta{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
				},
			}
			if err := cnt.Get(ctx, client.ObjectKey{Name: owner.Name, Namespace: om.Namespace}, obj); err != nil {
				log.Error(err, "failed to query the object", "apiVersion", owner.APIVersion, "kind", owner.Kind, "name", owner.Name, "namespace", om.Namespace)
				return o.ObjectMeta.Name, o.Kind, err
			}

			return rootOwner(ctx, cnt, obj)
		}
	}
	return o.ObjectMeta.Name, o.Kind, nil
}

// podMutator adds an annotation to every incoming pods
func (m *podMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if m.apmExists {
		return admission.Patched("")
	}

	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "Failed to decode the request for pod injection")
		return admission.Errored(http.StatusBadRequest, err)
	}

	inject := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInject, "true")
	if inject == "false" {
		return admission.Patched("")
	}

	var ns corev1.Namespace
	if err := m.client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &ns); err != nil {
		log.Error(err, "Failed to query the namespace before pod injection")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	dkName, ok := ns.Labels[mapper.InstanceLabel]
	if !ok {
		return admission.Errored(http.StatusBadRequest, fmt.Errorf("no DynaKube instance set for namespace: %s", req.Namespace))
	}
	var dk dynatracev1beta1.DynaKube
	if err := m.client.Get(ctx, client.ObjectKey{Name: dkName, Namespace: m.namespace}, &dk); k8serrors.IsNotFound(err) {
		template := "namespace '%s' is assigned to DynaKube instance '%s' but doesn't exist"
		m.recorder.Eventf(
			&dynatracev1beta1.DynaKube{ObjectMeta: v1.ObjectMeta{Name: "placeholder", Namespace: m.namespace}},
			corev1.EventTypeWarning,
			missingDynakubeEvent,
			template, req.Namespace, dkName)
		return admission.Errored(http.StatusBadRequest, fmt.Errorf(
			template, req.Namespace, dkName))
	} else if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if !dk.NeedAppInjection() {
		return admission.Patched("")
	}

	var initSecret corev1.Secret
	if err := m.apiReader.Get(ctx, client.ObjectKey{Name: dtwebhook.SecretConfigName, Namespace: ns.Name}, &initSecret); k8serrors.IsNotFound(err) {
		if _, err := initgeneration.NewInitGenerator(m.client, m.apiReader, m.namespace, log).GenerateForNamespace(ctx, dk, ns.Name); err != nil {
			log.Error(err, "Failed to create the init secret before pod injection")
			return admission.Errored(http.StatusBadRequest, err)
		}
	} else if err != nil {
		log.Error(err, "failed to query the init secret before pod injection")
		return admission.Errored(http.StatusBadRequest, err)
	}

	endpointGenerator := dtingestendpoint.NewEndpointGenerator(m.client, m.apiReader, m.namespace, log)

	var endpointSecret corev1.Secret
	if err := m.apiReader.Get(ctx, client.ObjectKey{Name: dtingestendpoint.SecretEndpointName, Namespace: ns.Name}, &endpointSecret); k8serrors.IsNotFound(err) {
		if _, err := endpointGenerator.GenerateForNamespace(ctx, dkName, ns.Name); err != nil {
			log.Error(err, "failed to create the data-ingest endpoint secret before pod injection")
			return admission.Errored(http.StatusBadRequest, err)
		}
	} else if err != nil {
		log.Error(err, "failed to query the data-ingest endpoint secret before pod injection")
		return admission.Errored(http.StatusBadRequest, err)
	}

	dataIngestFields, err := endpointGenerator.PrepareFields(ctx, &dk)
	if err != nil {
		log.Error(err, "failed to query the data-ingest endpoint secret before pod injection")
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Info("injecting into Pod", "name", pod.Name, "generatedName", pod.GenerateName, "namespace", req.Namespace)

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
					log.Info("instrumenting missing container", "name", c.Name)

					deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID, deploymentmetadata.DeploymentTypeApplicationMonitoring)
					updateContainer(c, &dk, pod, deploymentMetadata, dataIngestFields)

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
				log.Info("updating pod with missing containers")
				m.recorder.Eventf(&dk,
					corev1.EventTypeNormal,
					updatePodEvent,
					"Updating pod %s in namespace %s with missing containers", pod.GenerateName, pod.Namespace)
				return getResponseForPod(pod, &req)
			}
		}

		return admission.Patched("")
	}
	pod.Annotations[dtwebhook.AnnotationInjected] = "true"

	workloadName, workloadKind, workloadErr := rootOwnerPod(ctx, m.metaClient, pod, req.Namespace)
	if workloadErr != nil {
		return admission.Errored(http.StatusInternalServerError, workloadErr)
	}

	technologies := url.QueryEscape(kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all"))
	installPath := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)
	installerURL := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, "")
	failurePolicy := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")
	image := m.image

	dkVol := corev1.VolumeSource{}
	mode := ""
	if dk.NeedsCSIDriver() {
		dkVol.CSI = &corev1.CSIVolumeSource{
			Driver: dtcsi.DriverName,
		}
		mode = "provisioned"
	} else {
		dkVol.EmptyDir = &corev1.EmptyDirVolumeSource{}
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
		},
		corev1.Volume{
			Name: "data-ingest-endpoint",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dtingestendpoint.SecretEndpointName,
				},
			},
		},
		corev1.Volume{
			Name: "data-ingest-enrichment",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
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

	var deploymentMetadata *deploymentmetadata.DeploymentMetadata
	if dk.CloudNativeFullstackMode() {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(m.clusterID, deploymentmetadata.DeploymentTypeCloudNative)
	} else {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(m.clusterID, deploymentmetadata.DeploymentTypeApplicationMonitoring)
	}

	ic := corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
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
			{Name: "DT_WORKLOAD_KIND", Value: workloadKind},
			{Name: "DT_WORKLOAD_NAME", Value: workloadName},
		},
		SecurityContext: sc,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "oneagent-bin", MountPath: "/mnt/bin"},
			{Name: "oneagent-share", MountPath: "/mnt/share"},
			{Name: "oneagent-config", MountPath: "/mnt/config"},
			{Name: "data-ingest-enrichment", MountPath: "/var/lib/dynatrace/enrichment"},
		},
		Resources: *dk.InitResources(),
	}

	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]

		updateInstallContainer(&ic, i+1, c.Name, c.Image)

		updateContainer(c, &dk, pod, deploymentMetadata, dataIngestFields)
	}

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, ic)

	m.recorder.Eventf(&dk,
		corev1.EventTypeNormal,
		injectEvent,
		"Injecting the necessary info into pod %s in namespace %s", basePodName, ns.Name)
	return getResponseForPod(pod, &req)
}

// InjectClient injects the client
func (m *podMutator) InjectClient(c client.Client) error {
	m.client = c
	return nil
}

// InjectDecoder injects the decoder
func (m *podMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

// updateInstallContainer adds Container to list of Containers of Install Container
func updateInstallContainer(ic *corev1.Container, number int, name string, image string) {
	log.Info("updating install container with new container", "containerName", name, "containerImage", image)
	ic.Env = append(ic.Env,
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_NAME", number), Value: name},
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_IMAGE", number), Value: image})
}

// updateContainer sets missing preload Variables
func updateContainer(c *corev1.Container, oa *dynatracev1beta1.DynaKube,
	pod *corev1.Pod, deploymentMetadata *deploymentmetadata.DeploymentMetadata, dataIngestFields map[string]string) {

	log.Info("updating container with missing preload variables", "containerName", c.Name)
	installPath := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

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
		},
		corev1.VolumeMount{
			Name:      "data-ingest-endpoint",
			MountPath: "/var/lib/dynatrace/enrichment/endpoint",
		},
		corev1.VolumeMount{
			Name:      "data-ingest-enrichment",
			MountPath: "/var/lib/dynatrace/enrichment"})

	c.Env = append(c.Env,
		corev1.EnvVar{
			Name:  "LD_PRELOAD",
			Value: installPath + "/agent/lib64/liboneagentproc.so",
		},
		corev1.EnvVar{
			Name:  "DT_DEPLOYMENT_METADATA",
			Value: deploymentMetadata.AsString(),
		},
		corev1.EnvVar{
			Name:  dtingestendpoint.UrlSecretField,
			Value: dataIngestFields[dtingestendpoint.UrlSecretField],
		},
		corev1.EnvVar{
			Name:  dtingestendpoint.TokenSecretField,
			Value: dataIngestFields[dtingestendpoint.TokenSecretField],
		},
	)

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

// getResponseForPod tries to format pod as json
func getResponseForPod(pod *corev1.Pod, req *admission.Request) admission.Response {
	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
