package e2e

import (
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

// GenerateStatefulSet is used for testing purposes
func GenerateStatefulSet(name, namespace, feature, kubeSystemUUID string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name + "-" + feature,
			GenerateName:    "",
			Namespace:       namespace,
			SelfLink:        "",
			UID:             "",
			ResourceVersion: "",
			Generation:      0,
			CreationTimestamp: metav1.Time{
				Time: time.Time{},
			},
			DeletionTimestamp:          nil,
			DeletionGracePeriodSeconds: (*int64)(nil),
			Labels: map[string]string{
				"dynatrace.com/component":         feature,
				"operator.dynatrace.com/feature":  feature,
				"operator.dynatrace.com/instance": name,
			},
			Annotations: map[string]string{
				"internal.operator.dynatrace.com/template-hash": "3561195704",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "dynatrace.com/v1beta1",
					Kind:               "DynaKube",
					Name:               name,
					UID:                "",
					Controller:         pointer.Bool(true), //(*bool)(0xc0004cc83b),
					BlockOwnerDeletion: pointer.Bool(true), //(*bool)(0xc0004cc83a),
				},
			},
			Finalizers:    []string(nil),
			ClusterName:   "",
			ManagedFields: []metav1.ManagedFieldsEntry(nil),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: (*int32)(nil),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dynatrace.com/component":         feature,
					"operator.dynatrace.com/feature":  feature,
					"operator.dynatrace.com/instance": name,
				},
				MatchExpressions: nil,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "",
					GenerateName:    "",
					Namespace:       "",
					SelfLink:        "",
					UID:             "",
					ResourceVersion: "",
					Generation:      0,
					CreationTimestamp: metav1.Time{
						Time: time.Time{},
					},
					DeletionTimestamp:          nil,
					DeletionGracePeriodSeconds: (*int64)(nil),
					Labels: map[string]string{
						"dynatrace.com/component":         feature,
						"operator.dynatrace.com/feature":  feature,
						"operator.dynatrace.com/instance": name,
					},
					Annotations: map[string]string{
						"internal.operator.dynatrace.com/custom-properties-hash": "",
						"internal.operator.dynatrace.com/version":                "",
					},
					OwnerReferences: []metav1.OwnerReference(nil),
					Finalizers:      []string(nil),
					ClusterName:     "",
					ManagedFields:   []metav1.ManagedFieldsEntry(nil),
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "truststore-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: (*corev1.HostPathVolumeSource)(nil),
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium:    "",
									SizeLimit: nil,
								},
								GCEPersistentDisk:     (*corev1.GCEPersistentDiskVolumeSource)(nil),
								AWSElasticBlockStore:  (*corev1.AWSElasticBlockStoreVolumeSource)(nil),
								GitRepo:               (*corev1.GitRepoVolumeSource)(nil),
								Secret:                (*corev1.SecretVolumeSource)(nil),
								NFS:                   (*corev1.NFSVolumeSource)(nil),
								ISCSI:                 (*corev1.ISCSIVolumeSource)(nil),
								Glusterfs:             (*corev1.GlusterfsVolumeSource)(nil),
								PersistentVolumeClaim: (*corev1.PersistentVolumeClaimVolumeSource)(nil),
								RBD:                   (*corev1.RBDVolumeSource)(nil),
								FlexVolume:            (*corev1.FlexVolumeSource)(nil),
								Cinder:                (*corev1.CinderVolumeSource)(nil),
								CephFS:                (*corev1.CephFSVolumeSource)(nil),
								Flocker:               (*corev1.FlockerVolumeSource)(nil),
								DownwardAPI:           (*corev1.DownwardAPIVolumeSource)(nil),
								FC:                    (*corev1.FCVolumeSource)(nil),
								AzureFile:             (*corev1.AzureFileVolumeSource)(nil),
								ConfigMap:             (*corev1.ConfigMapVolumeSource)(nil),
								VsphereVolume:         (*corev1.VsphereVirtualDiskVolumeSource)(nil),
								Quobyte:               (*corev1.QuobyteVolumeSource)(nil),
								AzureDisk:             (*corev1.AzureDiskVolumeSource)(nil),
								PhotonPersistentDisk:  (*corev1.PhotonPersistentDiskVolumeSource)(nil),
								Projected:             (*corev1.ProjectedVolumeSource)(nil),
								PortworxVolume:        (*corev1.PortworxVolumeSource)(nil),
								ScaleIO:               (*corev1.ScaleIOVolumeSource)(nil),
								StorageOS:             (*corev1.StorageOSVolumeSource)(nil),
								CSI:                   (*corev1.CSIVolumeSource)(nil),
								Ephemeral:             (*corev1.EphemeralVolumeSource)(nil),
							},
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "certificate-loader",
							Image: "",
							Command: []string{
								"/bin/bash",
							},
							Args: []string{
								"-c",
								"/opt/dynatrace/gateway/k8scrt2jks.sh",
							},
							WorkingDir: "/var/lib/dynatrace/gateway",
							Ports:      []corev1.ContainerPort(nil),
							EnvFrom:    []corev1.EnvFromSource(nil),
							Env:        []corev1.EnvVar(nil),
							Resources: corev1.ResourceRequirements{
								Limits:   corev1.ResourceList(nil),
								Requests: corev1.ResourceList(nil),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "truststore-volume",
									ReadOnly:         false,
									MountPath:        "/var/lib/dynatrace/gateway/ssl",
									SubPath:          "",
									MountPropagation: (*corev1.MountPropagationMode)(nil),
									SubPathExpr:      "",
								},
							},
							VolumeDevices:            []corev1.VolumeDevice(nil),
							LivenessProbe:            (*corev1.Probe)(nil),
							ReadinessProbe:           (*corev1.Probe)(nil),
							StartupProbe:             (*corev1.Probe)(nil),
							Lifecycle:                (*corev1.Lifecycle)(nil),
							TerminationMessagePath:   "",
							TerminationMessagePolicy: "",
							ImagePullPolicy:          "Always",
							SecurityContext:          (*corev1.SecurityContext)(nil),
							Stdin:                    false,
							StdinOnce:                false,
							TTY:                      false,
						},
					},
					Containers: []corev1.Container{
						{
							Name:       feature,
							Image:      "",
							Command:    []string(nil),
							Args:       []string(nil),
							WorkingDir: "",
							Ports:      []corev1.ContainerPort(nil),
							EnvFrom:    []corev1.EnvFromSource(nil),
							Env: []corev1.EnvVar{
								{
									Name:      "DT_CAPABILITIES",
									Value:     "kubernetes_monitoring",
									ValueFrom: (*corev1.EnvVarSource)(nil),
								},
								{
									Name:      "DT_ID_SEED_NAMESPACE",
									Value:     namespace,
									ValueFrom: (*corev1.EnvVarSource)(nil),
								},
								{
									Name:      "DT_ID_SEED_K8S_CLUSTER_ID",
									Value:     kubeSystemUUID,
									ValueFrom: (*corev1.EnvVarSource)(nil),
								},
								{
									Name:      "DT_DEPLOYMENT_METADATA",
									Value:     "orchestration_tech=Operator-active_gate;script_version=snapshot;orchestrator_id=" + kubeSystemUUID,
									ValueFrom: (*corev1.EnvVarSource)(nil),
								},
							},
							Resources: corev1.ResourceRequirements{
								Limits:   corev1.ResourceList(nil),
								Requests: corev1.ResourceList(nil),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "truststore-volume",
									ReadOnly:         true,
									MountPath:        "/opt/dynatrace/gateway/jre/lib/security/cacerts",
									SubPath:          "k8s-local.jks",
									MountPropagation: (*corev1.MountPropagationMode)(nil),
									SubPathExpr:      "",
								},
							},
							VolumeDevices: []corev1.VolumeDevice(nil),
							LivenessProbe: (*corev1.Probe)(nil),
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/rest/health",
										Port: intstr.IntOrString{
											IntVal: 9999,
										},
										Scheme: "HTTPS",
									},
								},
								InitialDelaySeconds: 90,
								PeriodSeconds:       15,
								FailureThreshold:    3,
							},
							StartupProbe:             (*corev1.Probe)(nil),
							Lifecycle:                (*corev1.Lifecycle)(nil),
							TerminationMessagePath:   "",
							TerminationMessagePolicy: "",
							ImagePullPolicy:          "Always",
							SecurityContext:          (*corev1.SecurityContext)(nil),
							Stdin:                    false,
							StdinOnce:                false,
							TTY:                      false,
						},
					},
					EphemeralContainers:           []corev1.EphemeralContainer(nil),
					RestartPolicy:                 "",
					TerminationGracePeriodSeconds: (*int64)(nil),
					ActiveDeadlineSeconds:         (*int64)(nil),
					DNSPolicy:                     "",
					NodeSelector:                  map[string]string(nil),
					ServiceAccountName:            "dynatrace-kubernetes-monitoring",
					DeprecatedServiceAccount:      "",
					AutomountServiceAccountToken:  (*bool)(nil),
					NodeName:                      "",
					HostNetwork:                   false,
					HostPID:                       false,
					HostIPC:                       false,
					ShareProcessNamespace:         (*bool)(nil),
					SecurityContext:               (*corev1.PodSecurityContext)(nil),
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: name + "-pull-secret",
						},
					},
					Hostname:  "",
					Subdomain: "",
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/arch",
												Operator: "In",
												Values: []string{
													"amd64",
												},
											},
											{
												Key:      "kubernetes.io/os",
												Operator: "In",
												Values: []string{
													"linux",
												},
											},
										},
									},
								},
							},
						},
					},
					SchedulerName:             "",
					Tolerations:               []corev1.Toleration(nil),
					HostAliases:               []corev1.HostAlias(nil),
					PriorityClassName:         "",
					Priority:                  (*int32)(nil),
					DNSConfig:                 (*corev1.PodDNSConfig)(nil),
					ReadinessGates:            []corev1.PodReadinessGate(nil),
					RuntimeClassName:          (*string)(nil),
					EnableServiceLinks:        (*bool)(nil),
					PreemptionPolicy:          (*corev1.PreemptionPolicy)(nil),
					Overhead:                  corev1.ResourceList(nil),
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint(nil),
					SetHostnameAsFQDN:         (*bool)(nil),
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim(nil),
			ServiceName:          "",
			PodManagementPolicy:  "Parallel",
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type:          "",
				RollingUpdate: (*appsv1.RollingUpdateStatefulSetStrategy)(nil),
			},
			RevisionHistoryLimit: (*int32)(nil),
			MinReadySeconds:      0,
		},
		Status: appsv1.StatefulSetStatus{
			ObservedGeneration: 0,
			Replicas:           0,
			ReadyReplicas:      0,
			CurrentReplicas:    0,
			UpdatedReplicas:    0,
			CurrentRevision:    "",
			UpdateRevision:     "",
			CollisionCount:     (*int32)(nil),
			Conditions:         []appsv1.StatefulSetCondition(nil),
			AvailableReplicas:  0,
		},
	}
}
