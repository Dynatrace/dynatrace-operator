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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func findRootOwnerOfPod(ctx context.Context, clt client.Client, pod *corev1.Pod, namespace string) (string, string, error) {
	obj := &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: pod.APIVersion,
			Kind:       pod.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pod.ObjectMeta.Name,
			// pod.ObjectMeta.Namespace is empty yet
			Namespace:       namespace,
			OwnerReferences: pod.ObjectMeta.OwnerReferences,
		},
	}
	return findRootOwner(ctx, clt, obj)
}

func findRootOwner(ctx context.Context, clt client.Client, o *metav1.PartialObjectMetadata) (string, string, error) {
	if len(o.ObjectMeta.OwnerReferences) == 0 {
		kind := o.Kind
		if kind == "Pod" {
			kind = ""
		}
		return o.ObjectMeta.Name, kind, nil
	}

	om := o.ObjectMeta
	for _, owner := range om.OwnerReferences {
		if owner.Controller != nil && *owner.Controller && isWellKnownWorkload(owner) {
			obj := &metav1.PartialObjectMetadata{
				TypeMeta: metav1.TypeMeta{
					APIVersion: owner.APIVersion,
					Kind:       owner.Kind,
				},
			}
			if err := clt.Get(ctx, client.ObjectKey{Name: owner.Name, Namespace: om.Namespace}, obj); err != nil {
				log.Error(err, "failed to query the object", "apiVersion", owner.APIVersion, "kind", owner.Kind, "name", owner.Name, "namespace", om.Namespace)
				return o.ObjectMeta.Name, o.Kind, err
			}

			return findRootOwner(ctx, clt, obj)
		}
	}
	return o.ObjectMeta.Name, o.Kind, nil
}

func isWellKnownWorkload(ownerRef metav1.OwnerReference) bool {
	knownWorkloads := []metav1.TypeMeta{
		{Kind: "ReplicaSet", APIVersion: "apps/v1"},
		{Kind: "Deployment", APIVersion: "apps/v1"},
		{Kind: "ReplicationController", APIVersion: "v1"},
		{Kind: "StatefulSet", APIVersion: "apps/v1"},
		{Kind: "DaemonSet", APIVersion: "apps/v1"},
		{Kind: "Job", APIVersion: "batch/v1"},
		{Kind: "CronJob", APIVersion: "batch/v1"},
		{Kind: "DeploymentConfig", APIVersion: "apps.openshift.io/v1"},
	}

	for _, knownController := range knownWorkloads {
		if ownerRef.Kind == knownController.Kind &&
			ownerRef.APIVersion == knownController.APIVersion {
			return true
		}
	}
	return false
}

// podMutator adds an annotation to every incoming pods
func (m *podMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	emptyPatch := admission.Patched("")

	if m.apmExists {
		return emptyPatch
	}

	pod, rsp := m.getPod(req)
	if rsp != nil {
		return *rsp
	}

	injectionInfo := m.CreateInjectionInfo(pod)
	if !injectionInfo.anyEnabled() {
		return emptyPatch
	}

	ns, dkName, nsResponse, nsDone := m.getNsAndDkName(ctx, req)
	if nsDone {
		return nsResponse
	}

	dk, dkResponse, dkDone := m.getDk(ctx, req, dkName)
	if dkDone {
		return dkResponse
	}

	if !dk.NeedAppInjection() {
		return emptyPatch
	}

	secretResponse, secretDone := m.ensureInitSecret(ctx, ns, dk)
	if secretDone {
		return secretResponse
	}

	dataIngestFields, diResponse, diDone := m.ensureDataIngestSecret(ctx, ns, dkName, dk)
	if diDone {
		return diResponse
	}

	log.Info("injecting into Pod", "name", pod.Name, "generatedName", pod.GenerateName, "namespace", req.Namespace)

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	reinvocationResponse, reinvocationDone := m.ensureReinvocationPolicy(pod, dk, injectionInfo, dataIngestFields, req, emptyPatch)
	if reinvocationDone {
		return reinvocationResponse
	}

	pod.Annotations[dtwebhook.AnnotationDynatraceInjected] = injectionInfo.injectedAnnotation()

	workloadName, workloadKind, response, done := m.retrieveWorkload(ctx, req, injectionInfo, pod)
	if done {
		return response
	}

	technologies, installPath, installerURL, failurePolicy, image := m.getBasicData(pod)

	dkVol, mode := m.ensureDkVolume(dk)

	m.ensureInjectionConfigVolume(pod)

	m.ensureOneAgentVolumes(injectionInfo, pod, dkVol)

	m.ensureDataIngestVolumes(injectionInfo, pod)

	sc := m.getSecurityContext(pod)

	basePodName := m.getBasePodName(pod)

	deploymentMetadata := m.getDeploymentMetadata(dk)

	ic := m.createInstallInitContainerBase(image, pod, failurePolicy, basePodName, sc, dk)

	decorateInstallContainerWithOA(&ic, injectionInfo, technologies, installPath, installerURL, mode)
	decorateInstallContainerWithDI(&ic, injectionInfo, workloadKind, workloadName)

	updateContainers(pod, injectionInfo, &ic, dk, deploymentMetadata, dataIngestFields)

	pod.Spec.InitContainers = append(pod.Spec.InitContainers, ic)

	m.recorder.Eventf(&dk,
		corev1.EventTypeNormal,
		injectEvent,
		"Injecting the necessary info into pod %s in namespace %s", basePodName, ns.Name)

	return getResponseForPod(pod, &req)
}

func updateContainers(pod *corev1.Pod, injectionInfo *InjectionInfo, ic *corev1.Container, dk dynatracev1beta1.DynaKube, deploymentMetadata *deploymentmetadata.DeploymentMetadata, dataIngestFields map[string]string) {
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]

		if injectionInfo.enabled(OneAgent) {
			updateInstallContainerOA(ic, i+1, c.Name, c.Image)
			updateContainerOA(c, &dk, pod, deploymentMetadata, injectionInfo, dataIngestFields)
		}
		if injectionInfo.enabled(DataIngest) {
			updateContainerDI(c, &dk, pod, deploymentMetadata, injectionInfo, dataIngestFields)
		}
	}
}

func decorateInstallContainerWithDI(ic *corev1.Container, injectionInfo *InjectionInfo, workloadKind string, workloadName string) {
	const dataIngestEnabledEnvVarName = "DATA_INGEST_INJECTED"
	if injectionInfo.enabled(DataIngest) {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: "DT_WORKLOAD_KIND", Value: workloadKind},
			corev1.EnvVar{Name: "DT_WORKLOAD_NAME", Value: workloadName},
			corev1.EnvVar{Name: dataIngestEnabledEnvVarName, Value: "true"},
		)

		ic.VolumeMounts = append(ic.VolumeMounts, corev1.VolumeMount{
			Name:      "data-ingest-enrichment",
			MountPath: "/var/lib/dynatrace/enrichment"})
	} else {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: dataIngestEnabledEnvVarName, Value: "false"},
		)
	}
}

func decorateInstallContainerWithOA(ic *corev1.Container, injectionInfo *InjectionInfo, technologies string, installPath string, installerURL string, mode string) {
	const oneagentInjectedEnvVarName = "ONEAGENT_INJECTED"
	if injectionInfo.enabled(OneAgent) {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: "FLAVOR", Value: dtclient.FlavorMultidistro},
			corev1.EnvVar{Name: "TECHNOLOGIES", Value: technologies},
			corev1.EnvVar{Name: "INSTALLPATH", Value: installPath},
			corev1.EnvVar{Name: "INSTALLER_URL", Value: installerURL},
			corev1.EnvVar{Name: "MODE", Value: mode},
			corev1.EnvVar{Name: oneagentInjectedEnvVarName, Value: "true"},
		)

		ic.VolumeMounts = append(ic.VolumeMounts,
			corev1.VolumeMount{Name: "oneagent-bin", MountPath: "/mnt/bin"},
			corev1.VolumeMount{Name: "oneagent-share", MountPath: "/mnt/share"},
		)
	} else {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: oneagentInjectedEnvVarName, Value: "false"},
		)
	}
}

func (m *podMutator) createInstallInitContainerBase(image string, pod *corev1.Pod, failurePolicy string, basePodName string, sc *corev1.SecurityContext, dk dynatracev1beta1.DynaKube) corev1.Container {
	ic := corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/bin/env"},
		Args:            []string{"bash", "/mnt/config/init.sh"},
		Env: []corev1.EnvVar{
			{Name: "CONTAINERS_COUNT", Value: strconv.Itoa(len(pod.Spec.Containers))},
			{Name: "FAILURE_POLICY", Value: failurePolicy},
			{Name: "K8S_PODNAME", ValueFrom: fieldEnvVar("metadata.name")},
			{Name: "K8S_PODUID", ValueFrom: fieldEnvVar("metadata.uid")},
			{Name: "K8S_BASEPODNAME", Value: basePodName},
			{Name: "K8S_NAMESPACE", ValueFrom: fieldEnvVar("metadata.namespace")},
			{Name: "K8S_NODE_NAME", ValueFrom: fieldEnvVar("spec.nodeName")},
			//{Name: "DT_WORKLOAD_KIND", Value: workloadKind},
			//{Name: "DT_WORKLOAD_NAME", Value: workloadName},
		},
		SecurityContext: sc,
		VolumeMounts: []corev1.VolumeMount{
			{Name: "injection-config", MountPath: "/mnt/config"},
			//{Name: "data-ingest-enrichment", MountPath: "/var/lib/dynatrace/enrichment"},
		},
		Resources: *dk.InitResources(),
	}
	return ic
}

func (m *podMutator) getDeploymentMetadata(dk dynatracev1beta1.DynaKube) *deploymentmetadata.DeploymentMetadata {
	var deploymentMetadata *deploymentmetadata.DeploymentMetadata
	if dk.CloudNativeFullstackMode() {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(m.clusterID, deploymentmetadata.DeploymentTypeCloudNative)
	} else {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(m.clusterID, deploymentmetadata.DeploymentTypeApplicationMonitoring)
	}
	return deploymentMetadata
}

func (m *podMutator) getBasePodName(pod *corev1.Pod) string {
	basePodName := pod.GenerateName
	if basePodName == "" {
		basePodName = pod.Name
	}

	// Only include up to the last dash character, exclusive.
	if p := strings.LastIndex(basePodName, "-"); p != -1 {
		basePodName = basePodName[:p]
	}
	return basePodName
}

func (m *podMutator) getSecurityContext(pod *corev1.Pod) *corev1.SecurityContext {
	var sc *corev1.SecurityContext
	if pod.Spec.Containers[0].SecurityContext != nil {
		sc = pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}
	return sc
}

func (m *podMutator) ensureDataIngestVolumes(injectionInfo *InjectionInfo, pod *corev1.Pod) {
	if injectionInfo.enabled(DataIngest) {
		pod.Spec.Volumes = append(pod.Spec.Volumes,
			corev1.Volume{
				Name: "data-ingest-enrichment",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
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
		)
	}
}

func (m *podMutator) ensureOneAgentVolumes(injectionInfo *InjectionInfo, pod *corev1.Pod, dkVol corev1.VolumeSource) {
	if injectionInfo.enabled(OneAgent) {
		pod.Spec.Volumes = append(pod.Spec.Volumes,
			corev1.Volume{Name: "oneagent-bin", VolumeSource: dkVol},
			corev1.Volume{
				Name: "oneagent-share",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	}
}

func (m *podMutator) ensureInjectionConfigVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: "injection-config",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dtwebhook.SecretConfigName,
				},
			},
		},
	)
}

func (m *podMutator) getBasicData(pod *corev1.Pod) (string, string, string, string, string) {
	technologies := url.QueryEscape(kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all"))
	installPath := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)
	installerURL := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, "")
	failurePolicy := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")
	image := m.image
	return technologies, installPath, installerURL, failurePolicy, image
}

func (m *podMutator) ensureDkVolume(dk dynatracev1beta1.DynaKube) (corev1.VolumeSource, string) {
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
	return dkVol, mode
}

func (m *podMutator) retrieveWorkload(ctx context.Context, req admission.Request, injectionInfo *InjectionInfo, pod *corev1.Pod) (string, string, admission.Response, bool) {
	var workloadName, workloadKind string
	if injectionInfo.enabled(DataIngest) {
		var err error
		workloadName, workloadKind, err = findRootOwnerOfPod(ctx, m.metaClient, pod, req.Namespace)
		if err != nil {
			return "", "", admission.Errored(http.StatusInternalServerError, err), true
		}
	}
	return workloadName, workloadKind, admission.Response{}, false
}

func (m *podMutator) ensureReinvocationPolicy(pod *corev1.Pod, dk dynatracev1beta1.DynaKube, injectionInfo *InjectionInfo, dataIngestFields map[string]string, req admission.Request, emptyPatch admission.Response) (admission.Response, bool) {
	if len(pod.Annotations[dtwebhook.AnnotationDynatraceInjected]) > 0 {
		if dk.FeatureEnableWebhookReinvocationPolicy() {
			var needsUpdate = false
			var installContainer *corev1.Container
			for i := range pod.Spec.Containers {
				c := &pod.Spec.Containers[i]

				oaInjected := false
				if injectionInfo.enabled(OneAgent) {
					for _, e := range c.Env {
						if e.Name == "LD_PRELOAD" {
							oaInjected = true
							break
						}
					}
				}
				diInjected := false
				if injectionInfo.enabled(DataIngest) {
					for _, vm := range c.VolumeMounts {
						if vm.Name == "data-ingest-endpoint" {
							diInjected = true
							break
						}
					}
				}

				oaInjectionMissing := injectionInfo.enabled(OneAgent) && !oaInjected
				diInjectionMissing := injectionInfo.enabled(DataIngest) && !diInjected

				if oaInjectionMissing {
					// container does not have LD_PRELOAD set
					log.Info("instrumenting missing container", "name", c.Name)

					deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID, deploymentmetadata.DeploymentTypeApplicationMonitoring)

					updateContainerOA(c, &dk, pod, deploymentMetadata, injectionInfo, dataIngestFields)

					if installContainer == nil {
						for j := range pod.Spec.InitContainers {
							ic := &pod.Spec.InitContainers[j]

							if ic.Name == dtwebhook.InstallContainerName {
								installContainer = ic
								break
							}
						}
					}
					updateInstallContainerOA(installContainer, i+1, c.Name, c.Image)

					needsUpdate = true
				}

				if diInjectionMissing {
					log.Info("instrumenting missing container", "name", c.Name)

					deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID, deploymentmetadata.DeploymentTypeApplicationMonitoring)
					updateContainerDI(c, &dk, pod, deploymentMetadata, injectionInfo, dataIngestFields)

					needsUpdate = true
				}
			}

			if needsUpdate {
				log.Info("updating pod with missing containers")
				m.recorder.Eventf(&dk,
					corev1.EventTypeNormal,
					updatePodEvent,
					"Updating pod %s in namespace %s with missing containers", pod.GenerateName, pod.Namespace)
				return getResponseForPod(pod, &req), true
			}
		}

		return emptyPatch, true
	}
	return admission.Response{}, false
}

func (m *podMutator) getPod(req admission.Request) (*corev1.Pod, *admission.Response) {
	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		log.Error(err, "Failed to decode the request for pod injection")
		rsp := admission.Errored(http.StatusBadRequest, err)
		return nil, &rsp
	}
	return pod, nil
}

func (m *podMutator) getNsAndDkName(ctx context.Context, req admission.Request) (corev1.Namespace, string, admission.Response, bool) {
	var ns corev1.Namespace
	if err := m.client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &ns); err != nil {
		log.Error(err, "Failed to query the namespace before pod injection")
		return corev1.Namespace{}, "", admission.Errored(http.StatusInternalServerError, err), true
	}

	dkName, ok := ns.Labels[mapper.InstanceLabel]
	if !ok {
		return corev1.Namespace{}, "", admission.Errored(http.StatusBadRequest, fmt.Errorf("no DynaKube instance set for namespace: %s", req.Namespace)), true
	}
	return ns, dkName, admission.Response{}, false
}

func (m *podMutator) ensureDataIngestSecret(ctx context.Context, ns corev1.Namespace, dkName string, dk dynatracev1beta1.DynaKube) (map[string]string, admission.Response, bool) {
	endpointGenerator := dtingestendpoint.NewEndpointSecretGenerator(m.client, m.apiReader, m.namespace, log)

	var endpointSecret corev1.Secret
	if err := m.apiReader.Get(ctx, client.ObjectKey{Name: dtingestendpoint.SecretEndpointName, Namespace: ns.Name}, &endpointSecret); k8serrors.IsNotFound(err) {
		if _, err := endpointGenerator.GenerateForNamespace(ctx, dkName, ns.Name); err != nil {
			log.Error(err, "failed to create the data-ingest endpoint secret before pod injection")
			return nil, admission.Errored(http.StatusBadRequest, err), true
		}
	} else if err != nil {
		log.Error(err, "failed to query the data-ingest endpoint secret before pod injection")
		return nil, admission.Errored(http.StatusBadRequest, err), true
	}

	dataIngestFields, err := endpointGenerator.PrepareFields(ctx, &dk)
	if err != nil {
		log.Error(err, "failed to query the data-ingest endpoint secret before pod injection")
		return nil, admission.Errored(http.StatusBadRequest, err), true
	}
	return dataIngestFields, admission.Response{}, false
}

func (m *podMutator) getDk(ctx context.Context, req admission.Request, dkName string) (dynatracev1beta1.DynaKube, admission.Response, bool) {
	var dk dynatracev1beta1.DynaKube
	if err := m.client.Get(ctx, client.ObjectKey{Name: dkName, Namespace: m.namespace}, &dk); k8serrors.IsNotFound(err) {
		template := "namespace '%s' is assigned to DynaKube instance '%s' but doesn't exist"
		m.recorder.Eventf(
			&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "placeholder", Namespace: m.namespace}},
			corev1.EventTypeWarning,
			missingDynakubeEvent,
			template, req.Namespace, dkName)
		return dynatracev1beta1.DynaKube{}, admission.Errored(http.StatusBadRequest, fmt.Errorf(
			template, req.Namespace, dkName)), true
	} else if err != nil {
		return dynatracev1beta1.DynaKube{}, admission.Errored(http.StatusInternalServerError, err), true
	}
	return dk, admission.Response{}, false
}

func (m *podMutator) ensureInitSecret(ctx context.Context, ns corev1.Namespace, dk dynatracev1beta1.DynaKube) (admission.Response, bool) {
	var initSecret corev1.Secret
	if err := m.apiReader.Get(ctx, client.ObjectKey{Name: dtwebhook.SecretConfigName, Namespace: ns.Name}, &initSecret); k8serrors.IsNotFound(err) {
		if _, err := initgeneration.NewInitGenerator(m.client, m.apiReader, m.namespace, log).GenerateForNamespace(ctx, dk, ns.Name); err != nil {
			log.Error(err, "Failed to create the init secret before pod injection")
			return admission.Errored(http.StatusBadRequest, err), true
		}
	} else if err != nil {
		log.Error(err, "failed to query the init secret before pod injection")
		return admission.Errored(http.StatusBadRequest, err), true
	}
	return admission.Response{}, false
}

func (m *podMutator) CreateInjectionInfo(pod *corev1.Pod) *InjectionInfo {
	oneAgentInject := kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationOneAgentInject, true)
	dataIngestInject := kubeobjects.GetFieldBool(pod.Annotations, dtwebhook.AnnotationDataIngestInject, oneAgentInject)

	injectionInfo := NewInjectionInfo()
	if oneAgentInject {
		injectionInfo.add(NewFeature(OneAgent))
	}
	if dataIngestInject {
		injectionInfo.add(NewFeature(DataIngest))
	}
	return injectionInfo
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

// updateInstallContainerOA adds Container to list of Containers of Install Container
func updateInstallContainerOA(ic *corev1.Container, number int, name string, image string) {
	log.Info("updating install container with new container", "containerName", name, "containerImage", image)
	ic.Env = append(ic.Env,
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_NAME", number), Value: name},
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_IMAGE", number), Value: image})
}

// updateContainerOA sets missing preload Variables
func updateContainerOA(c *corev1.Container, oa *dynatracev1beta1.DynaKube, pod *corev1.Pod,
	deploymentMetadata *deploymentmetadata.DeploymentMetadata, injectionInfo *InjectionInfo, dataIngestFields map[string]string) {

	log.Info("updating container with missing preload variables", "containerName", c.Name)
	installPath := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	addMetadataIfMissing(c, deploymentMetadata)

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

func addMetadataIfMissing(c *corev1.Container, deploymentMetadata *deploymentmetadata.DeploymentMetadata) {
	const mtName = "DT_DEPLOYMENT_METADATA"

	for _, v := range c.Env {
		if v.Name == mtName {
			return
		}
	}

	c.Env = append(c.Env,
		corev1.EnvVar{
			Name:  mtName,
			Value: deploymentMetadata.AsString(),
		})
}

func updateContainerDI(c *corev1.Container, oa *dynatracev1beta1.DynaKube, pod *corev1.Pod,
	deploymentMetadata *deploymentmetadata.DeploymentMetadata, injectionInfo *InjectionInfo, dataIngestFields map[string]string) {

	log.Info("updating container with missing data ingest enrichment", "containerName", c.Name)

	addMetadataIfMissing(c, deploymentMetadata)

	c.VolumeMounts = append(c.VolumeMounts,
		corev1.VolumeMount{
			Name:      "data-ingest-enrichment",
			MountPath: "/var/lib/dynatrace/enrichment",
		},
		corev1.VolumeMount{
			Name:      "data-ingest-endpoint",
			MountPath: "/var/lib/dynatrace/enrichment/endpoint",
		},
	)

	c.Env = append(c.Env,
		corev1.EnvVar{
			Name:  dtingestendpoint.UrlSecretField,
			Value: dataIngestFields[dtingestendpoint.UrlSecretField],
		},
		corev1.EnvVar{
			Name:  dtingestendpoint.TokenSecretField,
			Value: dataIngestFields[dtingestendpoint.TokenSecretField],
		},
	)
}

// getResponseForPod tries to format pod as json
func getResponseForPod(pod *corev1.Pod, req *admission.Request) admission.Response {
	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}
