package mutation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/arch"
	dtcsi "github.com/Dynatrace/dynatrace-operator/src/controllers/csi"
	csivolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes"
	appvolumes "github.com/Dynatrace/dynatrace-operator/src/controllers/csi/driver/volumes/app"
	"github.com/Dynatrace/dynatrace-operator/src/controllers/dynakube/oneagent/daemonset"
	"github.com/Dynatrace/dynatrace-operator/src/deploymentmetadata"
	dtingestendpoint "github.com/Dynatrace/dynatrace-operator/src/ingestendpoint"
	"github.com/Dynatrace/dynatrace-operator/src/initgeneration"
	"github.com/Dynatrace/dynatrace-operator/src/kubeobjects"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/mapper"
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	dtwebhook "github.com/Dynatrace/dynatrace-operator/src/webhook"
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

const oneAgentCustomKeysPath = "/var/lib/dynatrace/oneagent/agent/customkeys"

var podLog = log.WithName("pod")

// AddPodMutationWebhookToManager adds the Webhook server to the Manager
func AddPodMutationWebhookToManager(mgr manager.Manager, ns string) error {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podLog.Info("no Pod name set for webhook container")
	}

	if err := registerInjectEndpoint(mgr, ns, podName); err != nil {
		return err
	}
	registerHealthzEndpoint(mgr)
	return nil
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
		podLog.Info("OneAgentAPM object detected - DynaKube webhook won't inject until the OneAgent Operator has been uninstalled")
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
	mgr.GetWebhookServer().Register("/livez", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

// podMutator injects the OneAgent into Pods
type podMutator struct {
	client         client.Client
	metaClient     client.Client
	apiReader      client.Reader
	decoder        *admission.Decoder
	image          string
	namespace      string
	apmExists      bool
	clusterID      string
	currentPodName string
	recorder       record.EventRecorder
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
				podLog.Error(err, "failed to query the object", "apiVersion", owner.APIVersion, "kind", owner.Kind, "name", owner.Name, "namespace", om.Namespace)
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
	m.currentPodName = pod.Name
	defer func() {
		m.currentPodName = ""
	}()

	injectionInfo := NewInjectionInfoForPod(pod)
	if !injectionInfo.anyEnabled() {
		return emptyPatch
	}

	ns, dkName, nsResponse := m.getNsAndDkName(ctx, req)
	if nsResponse != nil {
		return *nsResponse
	}

	dk, dkResponse := m.getDynakube(ctx, req, dkName)
	if dkResponse != nil {
		return *dkResponse
	}

	if dk.FeatureDisableMetadataEnrichment() {
		injectionInfo.features[DataIngest] = false
	}

	if !dk.NeedAppInjection() {
		return emptyPatch
	}

	secretResponse := m.ensureInitSecret(ctx, ns, dk)
	if secretResponse != nil {
		return *secretResponse
	}

	if injectionInfo.enabled(DataIngest) {
		err := m.ensureDataIngestSecret(ctx, ns, dkName)
		if err != nil {
			return silentErrorResponse(m.currentPodName, err)
		}
	}

	podLog.Info("injecting into Pod", "name", pod.Name, "generatedName", pod.GenerateName, "namespace", req.Namespace)

	response := m.handleAlreadyInjectedPod(pod, dk, injectionInfo, req)
	if response != nil {
		return *response
	}

	injectionInfo.fillAnnotations(pod)

	workloadName, workloadKind, workloadResponse := m.retrieveWorkload(ctx, req, injectionInfo, pod)
	if workloadResponse != nil {
		return *workloadResponse
	}

	flavor, technologies, installPath, installerURL, failurePolicy, image := m.getBasicData(pod)

	dkVol, mode := ensureDynakubeVolume(dk)

	setupInjectionConfigVolume(pod)
	setupOneAgentVolumes(injectionInfo, pod, dkVol)
	setupDataIngestVolumes(injectionInfo, pod)

	sc := getSecurityContext(pod)
	basePodName := getBasePodName(pod)
	deploymentMetadata := m.getDeploymentMetadata(dk)

	installContainer := createInstallInitContainerBase(image, pod, failurePolicy, basePodName, sc, dk)

	decorateInstallContainerWithOneAgent(&installContainer, injectionInfo, flavor, technologies, installPath, installerURL, mode)
	decorateInstallContainerWithDataIngest(&installContainer, injectionInfo, workloadKind, workloadName)

	updateContainers(pod, injectionInfo, &installContainer, dk, deploymentMetadata)

	addToInitContainers(pod, installContainer)

	m.recorder.Eventf(&dk,
		corev1.EventTypeNormal,
		injectEvent,
		"Injecting the necessary info into pod %s in namespace %s", basePodName, ns.Name)

	return getResponseForPod(pod, &req)
}

func (m *podMutator) handleAlreadyInjectedPod(pod *corev1.Pod, dk dynatracev1beta1.DynaKube, injectionInfo *InjectionInfo, req admission.Request) *admission.Response {
	// are there any injections already?
	if len(pod.Annotations[dtwebhook.AnnotationDynatraceInjected]) > 0 {
		if dk.FeatureEnableWebhookReinvocationPolicy() {
			rsp := m.applyReinvocationPolicy(pod, dk, injectionInfo, req)
			return &rsp
		}
		rsp := admission.Patched("")
		return &rsp
	}
	return nil
}

func addToInitContainers(pod *corev1.Pod, installContainer corev1.Container) {
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, installContainer)
}

func updateContainers(pod *corev1.Pod, injectionInfo *InjectionInfo, ic *corev1.Container, dk dynatracev1beta1.DynaKube, deploymentMetadata *deploymentmetadata.DeploymentMetadata) {
	for i := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]

		if injectionInfo.enabled(OneAgent) {
			updateInstallContainerOneAgent(ic, i+1, c.Name, c.Image)
			updateContainerOneAgent(c, &dk, pod, deploymentMetadata)
		}
		if injectionInfo.enabled(DataIngest) {
			updateContainerDataIngest(c, deploymentMetadata)
		}
	}
}

func decorateInstallContainerWithDataIngest(ic *corev1.Container, injectionInfo *InjectionInfo, workloadKind string, workloadName string) {
	if injectionInfo.enabled(DataIngest) {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: standalone.WorkloadKindEnv, Value: workloadKind},
			corev1.EnvVar{Name: standalone.WorkloadNameEnv, Value: workloadName},
			corev1.EnvVar{Name: standalone.DataIngestInjectedEnv, Value: "true"},
		)

		ic.VolumeMounts = append(ic.VolumeMounts, corev1.VolumeMount{
			Name:      dataIngestVolumeName,
			MountPath: standalone.EnrichmentPath})
	} else {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: standalone.DataIngestInjectedEnv, Value: "false"},
		)
	}
}

func decorateInstallContainerWithOneAgent(ic *corev1.Container, injectionInfo *InjectionInfo, flavor string, technologies string, installPath string, installerURL string, mode string) {
	if injectionInfo.enabled(OneAgent) {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: standalone.InstallerFlavorEnv, Value: flavor},
			corev1.EnvVar{Name: standalone.InstallerTechEnv, Value: technologies},
			corev1.EnvVar{Name: standalone.InstallPathEnv, Value: installPath},
			corev1.EnvVar{Name: standalone.InstallerUrlEnv, Value: installerURL},
			corev1.EnvVar{Name: standalone.ModeEnv, Value: mode},
			corev1.EnvVar{Name: standalone.OneAgentInjectedEnv, Value: "true"},
		)

		ic.VolumeMounts = append(ic.VolumeMounts,
			corev1.VolumeMount{Name: oneAgentBinVolumeName, MountPath: standalone.BinDirMount},
			corev1.VolumeMount{Name: oneAgentShareVolumeName, MountPath: standalone.ShareDirMount},
		)
	} else {
		ic.Env = append(ic.Env,
			corev1.EnvVar{Name: standalone.OneAgentInjectedEnv, Value: "false"},
		)
	}
}

func createInstallInitContainerBase(image string, pod *corev1.Pod, failurePolicy string, basePodName string, sc *corev1.SecurityContext, dk dynatracev1beta1.DynaKube) corev1.Container {
	ic := corev1.Container{
		Name:            dtwebhook.InstallContainerName,
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"init"},
		Env: []corev1.EnvVar{
			{Name: standalone.ContainerCountEnv, Value: strconv.Itoa(len(pod.Spec.Containers))},
			{Name: standalone.CanFailEnv, Value: failurePolicy},
			{Name: standalone.K8PodNameEnv, ValueFrom: fieldEnvVar("metadata.name")},
			{Name: standalone.K8PodUIDEnv, ValueFrom: fieldEnvVar("metadata.uid")},
			{Name: standalone.K8BasePodNameEnv, Value: basePodName},
			{Name: standalone.K8NamespaceEnv, ValueFrom: fieldEnvVar("metadata.namespace")},
			{Name: standalone.K8NodeNameEnv, ValueFrom: fieldEnvVar("spec.nodeName")},
		},
		SecurityContext: sc,
		VolumeMounts: []corev1.VolumeMount{
			{Name: injectionConfigVolumeName, MountPath: standalone.ConfigDirMount},
		},
		Resources: *dk.InitResources(),
	}
	return ic
}

func (m *podMutator) getDeploymentMetadata(dk dynatracev1beta1.DynaKube) *deploymentmetadata.DeploymentMetadata {
	var deploymentMetadata *deploymentmetadata.DeploymentMetadata
	if dk.CloudNativeFullstackMode() {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(m.clusterID, daemonset.DeploymentTypeCloudNative)
	} else {
		deploymentMetadata = deploymentmetadata.NewDeploymentMetadata(m.clusterID, daemonset.DeploymentTypeApplicationMonitoring)
	}
	return deploymentMetadata
}

func getBasePodName(pod *corev1.Pod) string {
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

func getSecurityContext(pod *corev1.Pod) *corev1.SecurityContext {
	var sc *corev1.SecurityContext
	if pod.Spec.Containers[0].SecurityContext != nil {
		sc = pod.Spec.Containers[0].SecurityContext.DeepCopy()
	}
	return sc
}

func setupDataIngestVolumes(injectionInfo *InjectionInfo, pod *corev1.Pod) {
	if !injectionInfo.enabled(DataIngest) {
		return
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: dataIngestVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		corev1.Volume{
			Name: dataIngestEndpointVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dtingestendpoint.SecretEndpointName,
				},
			},
		},
	)
}

func setupOneAgentVolumes(injectionInfo *InjectionInfo, pod *corev1.Pod, dkVol corev1.VolumeSource) {
	if !injectionInfo.enabled(OneAgent) {
		return
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{Name: oneAgentBinVolumeName, VolumeSource: dkVol},
		corev1.Volume{
			Name: oneAgentShareVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	)
}

func setupInjectionConfigVolume(pod *corev1.Pod) {
	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name: injectionConfigVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dtwebhook.SecretConfigName,
				},
			},
		},
	)
}

func (m *podMutator) getBasicData(pod *corev1.Pod) (
	flavor string,
	technologies string,
	installPath string,
	installerURL string,
	failurePolicy string,
	image string,
) {
	flavor = kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFlavor, arch.FlavorMultidistro)
	technologies = url.QueryEscape(kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationTechnologies, "all"))
	installPath = kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)
	installerURL = kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallerUrl, "")
	failurePolicy = kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationFailurePolicy, "silent")
	image = m.image
	return
}

func ensureDynakubeVolume(dk dynatracev1beta1.DynaKube) (corev1.VolumeSource, string) {
	dkVol := corev1.VolumeSource{}
	mode := ""
	if dk.NeedsCSIDriver() {
		dkVol.CSI = &corev1.CSIVolumeSource{
			Driver: dtcsi.DriverName,
			VolumeAttributes: map[string]string{
				csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
				csivolumes.CSIVolumeAttributeDynakubeField: dk.Name,
			},
		}
		mode = provisionedVolumeMode
	} else {
		dkVol.EmptyDir = &corev1.EmptyDirVolumeSource{}
		mode = installerVolumeMode
	}
	return dkVol, mode
}

func (m *podMutator) retrieveWorkload(ctx context.Context, req admission.Request, injectionInfo *InjectionInfo, pod *corev1.Pod) (string, string, *admission.Response) {
	var rsp admission.Response
	var workloadName, workloadKind string
	if injectionInfo.enabled(DataIngest) {
		var err error
		workloadName, workloadKind, err = findRootOwnerOfPod(ctx, m.metaClient, pod, req.Namespace)
		if err != nil {
			rsp = silentErrorResponse(m.currentPodName, err)
			return "", "", &rsp
		}
	}
	return workloadName, workloadKind, nil
}

func (m *podMutator) applyReinvocationPolicy(pod *corev1.Pod, dk dynatracev1beta1.DynaKube, injectionInfo *InjectionInfo, req admission.Request) admission.Response {
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
				if vm.Name == dataIngestEndpointVolumeName {
					diInjected = true
					break
				}
			}
		}

		oaInjectionMissing := injectionInfo.enabled(OneAgent) && !oaInjected
		diInjectionMissing := injectionInfo.enabled(DataIngest) && !diInjected

		if oaInjectionMissing {
			// container does not have LD_PRELOAD set
			podLog.Info("instrumenting missing container", "injectable", "oneagent", "name", c.Name)

			deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID, daemonset.DeploymentTypeApplicationMonitoring)

			updateContainerOneAgent(c, &dk, pod, deploymentMetadata)

			if installContainer == nil {
				for j := range pod.Spec.InitContainers {
					ic := &pod.Spec.InitContainers[j]

					if ic.Name == dtwebhook.InstallContainerName {
						installContainer = ic
						break
					}
				}
			}
			updateInstallContainerOneAgent(installContainer, i+1, c.Name, c.Image)

			needsUpdate = true
		}

		if diInjectionMissing {
			podLog.Info("instrumenting missing container", "injectable", "data-ingest", "name", c.Name)

			deploymentMetadata := deploymentmetadata.NewDeploymentMetadata(m.clusterID, daemonset.DeploymentTypeApplicationMonitoring)
			updateContainerDataIngest(c, deploymentMetadata)

			needsUpdate = true
		}
	}

	if needsUpdate {
		podLog.Info("updating pod with missing containers")
		m.recorder.Eventf(&dk,
			corev1.EventTypeNormal,
			updatePodEvent,
			"Updating pod %s in namespace %s with missing containers", pod.GenerateName, pod.Namespace)
		return getResponseForPod(pod, &req)
	}
	return admission.Patched("")
}

func (m *podMutator) getPod(req admission.Request) (*corev1.Pod, *admission.Response) {
	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		podLog.Error(err, "Failed to decode the request for pod injection")
		rsp := silentErrorResponse(req.Name, err)
		return nil, &rsp
	}
	return pod, nil
}

func (m *podMutator) getNsAndDkName(ctx context.Context, req admission.Request) (ns corev1.Namespace, dkName string, rspPtr *admission.Response) {
	var rsp admission.Response

	if err := m.client.Get(ctx, client.ObjectKey{Name: req.Namespace}, &ns); err != nil {
		podLog.Error(err, "Failed to query the namespace before pod injection")
		rsp = silentErrorResponse(m.currentPodName, err)
		return corev1.Namespace{}, "", &rsp
	}

	dkName, ok := ns.Labels[mapper.InstanceLabel]
	if !ok {
		if kubesystem.DeployedViaOLM() {
			rsp = admission.Patched("")
		} else {
			rsp = silentErrorResponse(m.currentPodName, fmt.Errorf("no DynaKube instance set for namespace: %s", req.Namespace))
		}
		return corev1.Namespace{}, "", &rsp
	}
	return ns, dkName, nil
}

func (m *podMutator) ensureDataIngestSecret(ctx context.Context, ns corev1.Namespace, dkName string) error {
	endpointGenerator := dtingestendpoint.NewEndpointSecretGenerator(m.client, m.apiReader, m.namespace)

	var endpointSecret corev1.Secret
	if err := m.apiReader.Get(ctx, client.ObjectKey{Name: dtingestendpoint.SecretEndpointName, Namespace: ns.Name}, &endpointSecret); k8serrors.IsNotFound(err) {
		if _, err := endpointGenerator.GenerateForNamespace(ctx, dkName, ns.Name); err != nil {
			podLog.Error(err, "failed to create the data-ingest endpoint secret before pod injection")
			return err
		}
	} else if err != nil {
		podLog.Error(err, "failed to query the data-ingest endpoint secret before pod injection")
		return err
	}

	return nil
}

func (m *podMutator) getDynakube(ctx context.Context, req admission.Request, dkName string) (dynatracev1beta1.DynaKube, *admission.Response) {
	var rsp admission.Response
	var dk dynatracev1beta1.DynaKube
	if err := m.client.Get(ctx, client.ObjectKey{Name: dkName, Namespace: m.namespace}, &dk); k8serrors.IsNotFound(err) {
		template := "namespace '%s' is assigned to DynaKube instance '%s' but doesn't exist"
		m.recorder.Eventf(
			&dynatracev1beta1.DynaKube{ObjectMeta: metav1.ObjectMeta{Name: "placeholder", Namespace: m.namespace}},
			corev1.EventTypeWarning,
			missingDynakubeEvent,
			template, req.Namespace, dkName)
		rsp = silentErrorResponse(m.currentPodName, fmt.Errorf(
			template, req.Namespace, dkName))
		return dynatracev1beta1.DynaKube{}, &rsp
	} else if err != nil {
		rsp = silentErrorResponse(m.currentPodName, err)
		return dynatracev1beta1.DynaKube{}, &rsp
	}
	return dk, nil
}

func (m *podMutator) ensureInitSecret(ctx context.Context, ns corev1.Namespace, dk dynatracev1beta1.DynaKube) *admission.Response {
	var initSecret corev1.Secret
	var rsp admission.Response

	if err := m.apiReader.Get(ctx, client.ObjectKey{Name: dtwebhook.SecretConfigName, Namespace: ns.Name}, &initSecret); k8serrors.IsNotFound(err) {
		if _, err := initgeneration.NewInitGenerator(m.client, m.apiReader, m.namespace).GenerateForNamespace(ctx, dk, ns.Name); err != nil {
			podLog.Error(err, "Failed to create the init secret before pod injection")
			rsp = silentErrorResponse(m.currentPodName, err)
			return &rsp
		}
	} else if err != nil {
		podLog.Error(err, "failed to query the init secret before pod injection")
		rsp = silentErrorResponse(m.currentPodName, err)
		return &rsp
	}
	return nil
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
func updateInstallContainerOneAgent(ic *corev1.Container, number int, name string, image string) {
	podLog.Info("updating install container with new container", "containerName", name, "containerImage", image)
	ic.Env = append(ic.Env,
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_NAME", number), Value: name},
		corev1.EnvVar{Name: fmt.Sprintf("CONTAINER_%d_IMAGE", number), Value: image})
}

// updateContainerOA sets missing preload Variables
func updateContainerOneAgent(c *corev1.Container, dk *dynatracev1beta1.DynaKube, pod *corev1.Pod, deploymentMetadata *deploymentmetadata.DeploymentMetadata) {

	podLog.Info("updating container with missing preload variables", "containerName", c.Name)
	installPath := kubeobjects.GetField(pod.Annotations, dtwebhook.AnnotationInstallPath, dtwebhook.DefaultInstallPath)

	addMetadataIfMissing(c, deploymentMetadata)

	c.VolumeMounts = append(c.VolumeMounts,
		corev1.VolumeMount{
			Name:      oneAgentShareVolumeName,
			MountPath: "/etc/ld.so.preload",
			SubPath:   "ld.so.preload",
		},
		corev1.VolumeMount{
			Name:      oneAgentBinVolumeName,
			MountPath: installPath,
		},
		corev1.VolumeMount{
			Name:      oneAgentShareVolumeName,
			MountPath: "/var/lib/dynatrace/oneagent/agent/config/container.conf",
			SubPath:   fmt.Sprintf(standalone.ContainerConfFilenameTemplate, c.Name),
		})
	if dk.HasActiveGateCaCert() {
		c.VolumeMounts = append(c.VolumeMounts,
			corev1.VolumeMount{
				Name:      oneAgentShareVolumeName,
				MountPath: filepath.Join(oneAgentCustomKeysPath, "custom.pem"),
				SubPath:   "custom.pem",
			})
	}

	c.Env = append(c.Env,
		corev1.EnvVar{
			Name:  "LD_PRELOAD",
			Value: installPath + "/agent/lib64/liboneagentproc.so",
		})

	if dk.Spec.Proxy != nil && (dk.Spec.Proxy.Value != "" || dk.Spec.Proxy.ValueFrom != "") {
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

	if dk.Spec.NetworkZone != "" {
		c.Env = append(c.Env, corev1.EnvVar{Name: "DT_NETWORK_ZONE", Value: dk.Spec.NetworkZone})
	}

}

func addMetadataIfMissing(c *corev1.Container, deploymentMetadata *deploymentmetadata.DeploymentMetadata) {
	for _, v := range c.Env {
		if v.Name == dynatraceMetadataEnvVarName {
			return
		}
	}

	c.Env = append(c.Env,
		corev1.EnvVar{
			Name:  dynatraceMetadataEnvVarName,
			Value: deploymentMetadata.AsString(),
		})
}

func updateContainerDataIngest(c *corev1.Container, deploymentMetadata *deploymentmetadata.DeploymentMetadata) {
	podLog.Info("updating container with missing data ingest enrichment", "containerName", c.Name)

	addMetadataIfMissing(c, deploymentMetadata)

	c.VolumeMounts = append(c.VolumeMounts,
		corev1.VolumeMount{
			Name:      dataIngestVolumeName,
			MountPath: standalone.EnrichmentPath,
		},
		corev1.VolumeMount{
			Name:      dataIngestEndpointVolumeName,
			MountPath: "/var/lib/dynatrace/enrichment/endpoint",
		},
	)
}

// getResponseForPod tries to format pod as json
func getResponseForPod(pod *corev1.Pod, req *admission.Request) admission.Response {
	marshaledPod, err := json.MarshalIndent(pod, "", "  ")
	if err != nil {
		return silentErrorResponse(pod.Name, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func fieldEnvVar(key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: key}}
}

func silentErrorResponse(podName string, err error) admission.Response {
	rsp := admission.Patched("")
	rsp.Result.Message = fmt.Sprintf("Failed to inject into pod: %s because %s", podName, err.Error())
	return rsp
}
