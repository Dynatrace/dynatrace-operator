package e2e

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// GenerateStatefulSet prepares an ActiveGate StatefulSet after a Reconciliation of the Dynakube with a specific feature enabled
func GenerateStatefulSet(name, namespace, feature, kubeSystemUUID string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-" + feature,
			Namespace: namespace,
			Labels: map[string]string{
				"dynatrace.com/component":         feature,
				"operator.dynatrace.com/feature":  feature,
				"operator.dynatrace.com/instance": name,
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "dynatrace.com/v1beta1",
					Kind:       "DynaKube",
					Name:       name,
				},
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"dynatrace.com/component":         feature,
					"operator.dynatrace.com/feature":  feature,
					"operator.dynatrace.com/instance": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"dynatrace.com/component":         feature,
						"operator.dynatrace.com/feature":  feature,
						"operator.dynatrace.com/instance": name,
					},
					Annotations: map[string]string{
						"internal.operator.dynatrace.com/custom-properties-hash": "",
						"internal.operator.dynatrace.com/version":                "",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "truststore-volume",
						},
					},
					InitContainers: []corev1.Container{
						{
							Name: "certificate-loader",
							Command: []string{
								"/bin/bash",
							},
							Args: []string{
								"-c",
								"/opt/dynatrace/gateway/k8scrt2jks.sh",
							},
							WorkingDir: "/var/lib/dynatrace/gateway",
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:             "truststore-volume",
									MountPath:        "/var/lib/dynatrace/gateway/ssl",
									MountPropagation: (*corev1.MountPropagationMode)(nil),
								},
							},
							ImagePullPolicy: "Always",
						},
					},
					Containers: []corev1.Container{
						{
							Name: feature,
							Env: []corev1.EnvVar{
								{
									Name:  "DT_CAPABILITIES",
									Value: "kubernetes_monitoring",
								},
								{
									Name:  "DT_ID_SEED_NAMESPACE",
									Value: namespace,
								},
								{
									Name:  "DT_ID_SEED_K8S_CLUSTER_ID",
									Value: kubeSystemUUID,
								},
								{
									Name:  "DT_DEPLOYMENT_METADATA",
									Value: "orchestration_tech=Operator-active_gate;script_version=snapshot;orchestrator_id=" + kubeSystemUUID,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "truststore-volume",
									ReadOnly:  true,
									MountPath: "/opt/dynatrace/gateway/jre/lib/security/cacerts",
									SubPath:   "k8s-local.jks",
								},
							},
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
							ImagePullPolicy: "Always",
						},
					},
					ServiceAccountName: "dynatrace-kubernetes-monitoring",
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: name + "-pull-secret",
						},
					},
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
				},
			},
			PodManagementPolicy: "Parallel",
		},
	}
}
