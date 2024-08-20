## DynaKube schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiUrl`|Dynatrace apiUrl, including the /api path at the end. For SaaS, set YOUR_ENVIRONMENT_ID to your environment ID. For Managed, change the apiUrl address.<br/>For instructions on how to determine the environment ID and how to configure the apiUrl address, see Environment ID (<https://www.dynatrace.com/support/help/get-started/monitoring-environment/environment-id>).|-|string|
|`customPullSecret`|Defines a custom pull secret in case you use a private registry when pulling images from the Dynatrace environment.<br/>To define a custom pull secret and learn about the expected behavior, see Configure customPullSecret<br/>(<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#custompullsecret>).|-|string|
|`dynatraceApiRequestThreshold`|Configuration for thresholding Dynatrace API requests.|15|integer|
|`enableIstio`|When enabled, and if Istio is installed on the Kubernetes environment, Dynatrace Operator will create the corresponding<br/>VirtualService and ServiceEntry objects to allow access to the Dynatrace Cluster from the OneAgent or ActiveGate.<br/>Disabled by default.|-|boolean|
|`networkZone`|Sets a network zone for the OneAgent and ActiveGate pods.|-|string|
|`proxy`|Set custom proxy settings either directly or from a secret with the field proxy.<br/>Note: Applies to Dynatrace Operator, ActiveGate, and OneAgents.|-|object|
|`skipCertCheck`|Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster.<br/>Set to true if you want to skip certification validation checks.|-|boolean|
|`tokens`|Name of the secret holding the tokens used for connecting to Dynatrace.|-|string|
|`trustedCAs`|Adds custom RootCAs from a configmap. Put the certificate under certs within your configmap.<br/>Note: Applies to Dynatrace Operator, OneAgent and ActiveGate.|-|string|

### .spec.oneAgent

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`hostGroup`|Sets a host group for OneAgent.|-|string|

### .spec.activeGate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the ActiveGate pods|-|object|
|`capabilities`|Activegate capabilities enabled (routing, kubernetes-monitoring, metrics-ingest, dynatrace-api)|-|array|
|`customProperties`|Add a custom properties file by providing it as a value or reference it from a secret<br/>If referenced from a secret, make sure the key is called 'customProperties'|-|object|
|`dnsPolicy`|Sets DNS Policy for the ActiveGate pods|-|string|
|`env`|List of environment variables to set for the ActiveGate|-|array|
|`group`|Set activation group for ActiveGate|-|string|
|`image`|The ActiveGate container image. Defaults to the latest ActiveGate image provided by the registry on the tenant|-|string|
|`labels`|Adds additional labels for the ActiveGate pods|-|object|
|`nodeSelector`|Node selector to control the selection of nodes|-|object|
|`priorityClassName`|If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that<br/>name. If not specified the setting will be removed from the StatefulSet.|-|string|
|`replicas`|Amount of replicas for your ActiveGates|1|integer|
|`resources`|Define resources requests and limits for single ActiveGate pods|-|object|
|`tlsSecretName`|The name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used.<br/>server.p12: certificate+key pair in pkcs12 format<br/>password: passphrase to read server.p12|-|string|
|`tolerations`|Set tolerations for the ActiveGate pods|-|array|
|`topologySpreadConstraints`|Adds TopologySpreadConstraints for the ActiveGate pods|-|array|

### .spec.metadataEnrichment

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`enabled`|Enables MetadataEnrichment, `true` by default.|True|boolean|
|`namespaceSelector`|The namespaces where you want Dynatrace Operator to inject enrichment.|-|object|

### .spec.extensions.prometheus

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`enabled`||-|boolean|

### .spec.oneAgent.hostMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer.<br/>For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>).<br/>For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>).<br/>Enabled by default.|True|boolean|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR.<br/>Note: resource.requests shows the values needed to run; resource.limits shows the maximum limits for the pod.|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set.<br/>For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|`version`|The OneAgent version to be used.|-|string|

### .spec.oneAgent.classicFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer.<br/>For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>).<br/>For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>).<br/>Enabled by default.|True|boolean|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR.<br/>Note: resource.requests shows the values needed to run; resource.limits shows the maximum limits for the pod.|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set.<br/>For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|`version`|The OneAgent version to be used.|-|string|

### .spec.oneAgent.cloudNativeFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer.<br/>For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>).<br/>For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>).<br/>Enabled by default.|True|boolean|
|`codeModulesImage`|The OneAgent image that is used to inject into Pods.|-|string|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|`initResources`|Define resources requests and limits for the initContainer. For details, see Managing resources for containers<br/>(<https://kubernetes.io/docs/concepts/configuration/manage-resources-containers>).|-|object|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`namespaceSelector`|Applicable only for applicationMonitoring or cloudNativeFullStack configuration types. The namespaces where you want Dynatrace Operator to inject.<br/>For more information, see Configure monitoring for namespaces and pods (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#annotate>).|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR.<br/>Note: resource.requests shows the values needed to run; resource.limits shows the maximum limits for the pod.|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set.<br/>For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|`version`|The OneAgent version to be used.|-|string|

### .spec.oneAgent.applicationMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`codeModulesImage`|The OneAgent image that is used to inject into Pods.|-|string|
|`initResources`|Define resources requests and limits for the initContainer. For details, see Managing resources for containers<br/>(<https://kubernetes.io/docs/concepts/configuration/manage-resources-containers>).|-|object|
|`namespaceSelector`|Applicable only for applicationMonitoring or cloudNativeFullStack configuration types. The namespaces where you want Dynatrace Operator to inject.<br/>For more information, see Configure monitoring for namespaces and pods (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#annotate>).|-|object|
|`useCSIDriver`|Set if you want to use the CSIDriver. Don't enable it if you do not have access to Kubernetes nodes or if you lack privileges.|False|boolean|
|`version`|The OneAgent version to be used.|-|string|

### .spec.templates.openTelemetryCollector

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the OtelCollector pods|-|object|
|`labels`|Adds additional labels for the OtelCollector pods|-|object|
|`replicas`|Number of replicas for your OtelCollector|1|integer|
|`resources`|Define resources' requests and limits for single OtelCollector pod|-|object|
|`tlsRefName`||-|string|
|`tolerations`|Set tolerations for the OtelCollector pods|-|array|
|`topologySpreadConstraints`|Adds TopologySpreadConstraints for the OtelCollector pods|-|array|

### .spec.templates.extensionExecutionController

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the ExtensionExecutionController pods|-|object|
|`customConfig`|Defines name of ConfigMap containing custom configuration file|-|string|
|`labels`|Adds additional labels for the ExtensionExecutionController pods|-|object|
|`persistentVolumeClaimRetentionPolicy`|Determines retention policy|-|string|
|`resources`|Define resources' requests and limits for single ExtensionExecutionController pod|-|object|
|`tlsRefName`||-|string|
|`tolerations`|Set tolerations for the ExtensionExecutionController pods|-|array|
|`topologySpreadConstraints`|Adds TopologySpreadConstraints for the ExtensionExecutionController pods|-|array|

### .spec.templates.openTelemetryCollector.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the image to use|-|string|

### .spec.templates.extensionExecutionController.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the image to use|-|string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`accessModes`|accessModes contains the desired access modes the volume should have.<br/>More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1|-|array|
|`dataSource`|dataSource field can be used to specify either:<br/>* An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot)<br/>* An existing PVC (PersistentVolumeClaim)<br/>If the provisioner or an external controller can support the specified data source,<br/>it will create a new volume based on the contents of the specified data source.|-|object|
|`resources`|resources represents the minimum resources the volume should have.<br/>If RecoverVolumeExpansionFailure feature is enabled users are allowed to specify resource requirements<br/>that are lower than previous value but must still be higher than capacity recorded in the<br/>status field of the claim.<br/>More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources|-|object|
|`selector`|selector is a label query over volumes to consider for binding.|-|object|
|`storageClassName`|storageClassName is the name of the StorageClass required by the claim.<br/>More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1|-|string|
|`volumeAttributesClassName`|volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim.<br/>If specified, the CSI driver will create or update the volume with the attributes defined<br/>in the corresponding VolumeAttributesClass. This has a different purpose than storageClassName,<br/>it can be changed after the claim is created. An empty string value means that no VolumeAttributesClass<br/>will be applied to the claim but it's not allowed to reset this field to empty string once it is set.|-|string|
|`volumeMode`|volumeMode defines what type of volume is required by the claim.<br/>Value of Filesystem is implied when not included in claim spec.|-|string|
|`volumeName`|volumeName is the binding reference to the PersistentVolume backing this claim.|-|string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim.dataSourceRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiGroup`|APIGroup is the group for the resource being referenced.<br/>If APIGroup is not specified, the specified Kind must be in the core API group.<br/>For any other third-party types, APIGroup is required.|-|string|
|`kind`|Kind is the type of resource being referenced|-|string|
|`name`|Name is the name of resource being referenced|-|string|
|`namespace`|Namespace is the namespace of resource being referenced<br/>Note that when a namespace is specified, a gateway.networking.k8s.io/ReferenceGrant object is required in the referent namespace to allow that namespace's owner to accept the reference. See the ReferenceGrant documentation for details.<br/>(Alpha) This field requires the CrossNamespaceVolumeDataSource feature gate to be enabled.|-|string|
