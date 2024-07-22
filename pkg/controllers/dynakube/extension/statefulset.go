package extension

import (
	"fmt"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/conditions"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/hasher"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/labels"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/node"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubeobjects/statefulset"
	maputils "github.com/Dynatrace/dynatrace-operator/pkg/util/map"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/prioritymap"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	statefulsetName                  = "dynatrace-extensions-controller"
	runtimePersistentVolumeClaimName = statefulsetName + "-runtime"
	containerName                    = "extensions-controller"
	collectorPort                    = int32(14599)

	envTenantId                     = "TenantId"
	envServerUrl                    = "ServerUrl"
	envEecTokenPath                 = "EecTokenPath"
	envEecIngestPort                = "EecIngestPort"
	envExtensionsConfPathName       = "ExtensionsConfPath"
	envExtensionsConfPath           = "/opt/dynatrace/remotepluginmodule/agent/conf/extensions.conf"
	envExtensionsModuleExecPathName = "ExtensionsModuleExecPath"
	envExtensionsModuleExecPath     = "/opt/dynatrace/remotepluginmodule/agent/lib64/extensionsmodule"
	envDsInstallDirName             = "DsInstallDir"
	envDsInstallDir                 = "/opt/dynatrace/remotepluginmodule/agent/datasources"
	envK8sClusterId                 = "K8sClusterId"

	eecTokenVolumeName      = "eec-token"
	eecTokenMountPath       = "/var/lib/dynatrace/remotepluginmodule/secrets/ag"
	eecFile                 = "eec.token"
	logVolumeName           = "log"
	logMountPath            = "/var/lib/dynatrace/remotepluginmodule/log"
	runtimeVolumeName       = "agent-runtime"
	runtimeMountPath        = "/var/lib/dynatrace/remotepluginmodule/agent/runtime"
	configurationVolumeName = "runtime-configuration"
	configurationMountPath  = "/var/lib/dynatrace/remotepluginmodule/agent/conf/runtime"
)

func (r *reconciler) reconcileStatefulset(ctx context.Context) error {
	if !r.dk.PrometheusEnabled() {
		if meta.FindStatusCondition(*r.dk.Conditions(), extensionsControllerStatefulSetConditionType) == nil {
			return nil
		}
		defer meta.RemoveStatusCondition(r.dk.Conditions(), extensionsControllerStatefulSetConditionType)

		sts, err := statefulset.Build(r.dk, statefulsetName)
		if err != nil {
			log.Error(err, "could not build "+statefulsetName+" during cleanup")

			return err
		}

		err = statefulset.Query(r.client, r.apiReader, log).Delete(ctx, sts)

		if err != nil {
			log.Error(err, "failed to clean up "+statefulsetName+" statufulset")

			return nil
		}

		return nil
	}

	if r.dk.Status.ActiveGate.ConnectionInfoStatus.TenantUUID == "" {
		conditions.SetStatefulSetOutdated(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, statefulsetName)
		return errors.New("tenantUUID unknown")
	}

	return r.createOrUpdateStatefulset(ctx)
}

func (r *reconciler) createOrUpdateStatefulset(ctx context.Context) error {
	desiredSts, err := statefulset.Build(r.dk, statefulsetName,
		setReplicas(),
		setPodManagementPolicy(),
		setLabels(r.dk.Spec.Templates.ExtensionExecutionController.Labels, r.dk.Name),
		setAnnotations(r.dk.Spec.Templates.ExtensionExecutionController.Annotations),
		setAffinity(),
		setTolerations(r.dk.Spec.Templates.ExtensionExecutionController.Tolerations),
		setTopologySpreadConstraints(r.dk.Spec.Templates.ExtensionExecutionController.TopologySpreadConstraints, r.dk.Name),
		setTlsRef(r.dk.Spec.Templates.ExtensionExecutionController.TlsRefName),
		setImagePullSecrets(r.dk.ImagePullSecretReferences()),
		setVolumes(r.dk.Name, r.dk.Spec.Templates.ExtensionExecutionController.PersistentVolumeClaim),
		setContainer(),
		setContainerImage(r.dk.Spec.Templates.ExtensionExecutionController.ImageRef.Repository, r.dk.Spec.Templates.ExtensionExecutionController.ImageRef.Tag),
		setContainerResources(r.dk.Spec.Templates.ExtensionExecutionController.Resources),
		setContainerEnvs(r.dk.Name, r.dk.Namespace, r.dk.Status.ActiveGate.ConnectionInfoStatus.TenantUUID, "activegate", r.dk.Status.KubeSystemUUID),
		setContainerVolumeMounts(),
	)

	if err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)
		return err
	}

	if err := setHash(desiredSts); err != nil {
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)
		return err
	}

	_, err = statefulset.Query(r.client, r.apiReader, log).WithOwner(r.dk).CreateOrUpdate(ctx, desiredSts)
	if err != nil {
		log.Info("failed to create/update " + statefulsetName + " statefulset")
		conditions.SetKubeApiError(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, err)

		return err
	}

	conditions.SetStatefulSetCreated(r.dk.Conditions(), extensionsControllerStatefulSetConditionType, desiredSts.Name)

	return nil
}

func setReplicas() func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		replicas := int32(1)
		o.Spec.Replicas = &replicas
	}
}

func setPodManagementPolicy() func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.PodManagementPolicy = appsv1.ParallelPodManagement
	}
}

func setLabels(labels map[string]string, dynakubeName string) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		appLabels := buildAppLabels(dynakubeName)
		o.ObjectMeta.Labels = appLabels.BuildLabels()
		o.Spec.Selector = &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()}
		o.Spec.Template.ObjectMeta.Labels = maputils.MergeMap(labels, appLabels.BuildLabels())
	}
}

func buildAppLabels(dynakubeName string) *labels.AppLabels {
	// version := statefulSetBuilder.dynakube.Status.ActiveGate.Version
	version := "0.0.0"

	return labels.NewAppLabels(labels.ExtensionComponentLabel, dynakubeName, labels.ExtensionComponentLabel, version)
}

func setAnnotations(annotations map[string]string) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.ObjectMeta.Annotations = maputils.MergeMap(o.ObjectMeta.Annotations, annotations)
		o.Spec.Template.ObjectMeta.Annotations = maputils.MergeMap(o.Spec.Template.ObjectMeta.Annotations, annotations)
	}
}

func setAffinity() func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: node.AffinityNodeRequirementForSupportedArches(),
						},
					},
				},
			},
		}
	}
}

func setTolerations(tolerations []corev1.Toleration) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Tolerations = tolerations
	}
}

func setTopologySpreadConstraints(topologySpreadConstraints []corev1.TopologySpreadConstraint, dynakubeName string) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		if len(topologySpreadConstraints) > 0 {
			o.Spec.Template.Spec.TopologySpreadConstraints = topologySpreadConstraints
		} else {
			appLabels := buildAppLabels(dynakubeName)

			o.Spec.Template.Spec.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "topology.kubernetes.io/zone",
					WhenUnsatisfiable: "ScheduleAnyway",
					LabelSelector:     &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
				},
				{
					MaxSkew:           1,
					TopologyKey:       "kubernetes.io/hostname",
					WhenUnsatisfiable: "DoNotSchedule",
					LabelSelector:     &metav1.LabelSelector{MatchLabels: appLabels.BuildMatchLabels()},
				},
			}
		}
	}
}

func setTlsRef(tlsRefName string) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {

	}
}

func setImagePullSecrets(imagePullSecrets []corev1.LocalObjectReference) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.ImagePullSecrets = imagePullSecrets
	}
}

func setContainerImage(repository string, tag string) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Containers[0].Image = repository + ":" + tag
	}
}

func setContainerResources(resources corev1.ResourceRequirements) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Containers[0].Resources = resources
	}
}

func setContainer() func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:            containerName,
				ImagePullPolicy: corev1.PullAlways,
				ReadinessProbe: &corev1.Probe{
					ProbeHandler: corev1.ProbeHandler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/readyz",
							// Port:   intstr.IntOrString{StrVal: "collector-com"},
							Port:   intstr.IntOrString{IntVal: collectorPort},
							Scheme: "HTTP",
						},
					},
					InitialDelaySeconds: 15,
					PeriodSeconds:       15,
					FailureThreshold:    3,
					TimeoutSeconds:      2,
					SuccessThreshold:    1,
				},
				SecurityContext: buildSecurityContext(),
				Ports: []corev1.ContainerPort{
					{
						Name:          "collector-com",
						ContainerPort: collectorPort,
					},
				},
			},
		}
	}
}

func buildSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func setContainerEnvs(dynakubeName, namespaceName, tenantId, activeGateName, kubeSystemUid string) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		envMap := prioritymap.New(prioritymap.WithPriority(prioritymap.DefaultPriority))
		prioritymap.Append(envMap, []corev1.EnvVar{
			{Name: envTenantId, Value: tenantId},
			{Name: envServerUrl, Value: dynakubeName + "-" + activeGateName + "." + namespaceName + ".svc.cluster.local:443"},
			{Name: envEecTokenPath, Value: eecTokenMountPath + "/" + eecFile},
			{Name: envEecIngestPort, Value: fmt.Sprint(collectorPort)},
			{Name: envExtensionsConfPathName, Value: envExtensionsConfPath},
			{Name: envExtensionsModuleExecPathName, Value: envExtensionsModuleExecPath},
			{Name: envDsInstallDirName, Value: envDsInstallDir},
			{Name: envK8sClusterId, Value: kubeSystemUid},
		})
		o.Spec.Template.Spec.Containers[0].Env = envMap.AsEnvVars()
	}
}

func setContainerVolumeMounts() func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		o.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{

			{
				Name:      eecTokenVolumeName,
				MountPath: eecTokenMountPath,
				ReadOnly:  true,
			},
			{
				Name:      logVolumeName,
				MountPath: logMountPath,
				ReadOnly:  false,
			},
			{
				Name:      runtimeVolumeName,
				MountPath: runtimeMountPath,
				ReadOnly:  false,
			},
			{
				Name:      configurationVolumeName,
				MountPath: configurationMountPath,
				ReadOnly:  true,
			},
		}
	}
}

func setVolumes(dynakubeName string, claim *corev1.PersistentVolumeClaimSpec) func(o *appsv1.StatefulSet) {
	return func(o *appsv1.StatefulSet) {
		mode := int32(420)
		o.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: eecTokenVolumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  dynakubeName + secretSuffix,
						DefaultMode: &mode,
						Items: []corev1.KeyToPath{
							{
								Key:  EecTokenSecretKey,
								Path: eecFile,
								Mode: &mode,
							},
						},
					},
				},
			},
			{
				Name: logVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			{
				/*
				   - name: runtime-configuration
				     configMap:
				       name: eec-runtime-configuration
				*/
				Name: configurationVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}

		if claim == nil {
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})

		} else {
			o.Spec.Template.Spec.Volumes = append(o.Spec.Template.Spec.Volumes, corev1.Volume{
				Name: runtimeVolumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: runtimePersistentVolumeClaimName,
					},
				},
			})
		}
	}
}

func setHash(o *appsv1.StatefulSet) error {
	hash, err := hasher.GenerateHash(o)
	if err != nil {
		return errors.WithStack(err)
	}

	o.ObjectMeta.Annotations[hasher.AnnotationHash] = hash

	return nil
}
