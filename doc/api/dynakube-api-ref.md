## DynaKube schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiUrl`|Dynatrace apiUrl, including the /api path at the end.|-|string|
|`customPullSecret`|Defines a custom pull secret in case you use a private registry when pulling images from the...|-|string|
|`dynatraceApiRequestThreshold`|Configuration for thresholding Dynatrace API requests.|-|integer|
|`enableIstio`|When enabled, and if Istio is installed on the Kubernetes environment, Dynatrace Operator will...|-|boolean|
|`extensions`|When an (empty) ExtensionsSpec is provided, the extensions related components (extensions...|-|object|
|`kspm`|General configuration about the KSPM feature.|-|object|
|`networkZone`|Sets a network zone for the OneAgent and ActiveGate pods.|-|string|
|`proxy`|Set custom proxy settings either directly or from a secret with the field proxy.|-|object|
|`skipCertCheck`|Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster.|-|boolean|
|`tokens`|Name of the secret holding the tokens used for connecting to Dynatrace.|-|string|
|`trustedCAs`|Adds custom RootCAs from a configmap. Put the certificate under certs within your configmap.|-|string|

### .spec.oneAgent

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`hostGroup`|Sets a host group for OneAgent.|-|string|

### .spec.activeGate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the ActiveGate pods|-|object|
|`capabilities`|Activegate capabilities enabled (routing, kubernetes-monitoring, metrics-ingest, dynatrace-api)|-|array|
|`customProperties`|Add a custom properties file by providing it as a value or reference it from a secret<br/>If referenced...|-|object|
|`dnsPolicy`|Sets DNS Policy for the ActiveGate pods|-|string|
|`env`|List of environment variables to set for the ActiveGate|-|array|
|`group`|Set activation group for ActiveGate|-|string|
|`image`|The ActiveGate container image.|-|string|
|`labels`|Adds additional labels for the ActiveGate pods|-|object|
|`nodeSelector`|Node selector to control the selection of nodes|-|object|
|`priorityClassName`|If specified, indicates the pod's priority.|-|string|
|`replicas`|Amount of replicas for your ActiveGates|-|integer|
|`resources`|Define resources requests and limits for single ActiveGate pods|-|object|
|`terminationGracePeriodSeconds`|Configures the terminationGracePeriodSeconds parameter of the ActiveGate pod.|-|integer|
|`tlsSecretName`|The name of a secret containing ActiveGate TLS cert+key and password.|-|string|
|`tolerations`|Set tolerations for the ActiveGate pods|-|array|
|`topologySpreadConstraints`|Adds TopologySpreadConstraints for the ActiveGate pods|-|array|
|`useEphemeralVolume`|UseEphemeralVolume|-|boolean|

### .spec.logMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`ingestRuleMatchers`||-|array|

### .spec.telemetryIngest

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`protocols`||-|array|
|`serviceName`||-|string|
|`tlsRefName`||-|string|

### .spec.metadataEnrichment

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`enabled`|Enables MetadataEnrichment, `false` by default.|-|boolean|
|`namespaceSelector`|The namespaces where you want Dynatrace Operator to inject enrichment.|-|object|

### .spec.oneAgent.hostMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer.|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.|-|boolean|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent image. Defaults to the latest image from the Dynatrace cluster.|-|string|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container.|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set.|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`storageHostPath`|StorageHostPath is the writable directory on the host filesystem where OneAgent configurations will...|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet.|-|array|
|`version`|Use a specific OneAgent version. Defaults to the latest version from the Dynatrace cluster.|-|string|

### .spec.templates.logMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom annotations to the LogMonitoring pods|-|object|
|`args`|Set additional arguments to the LogMonitoring main container|-|array|
|`dnsPolicy`|Sets DNS Policy for the LogMonitoring pods|-|string|
|`labels`|Add custom labels to the LogMonitoring pods|-|object|
|`nodeSelector`|Node selector to control the selection of nodes for the LogMonitoring pods|-|object|
|`priorityClassName`|Assign a priority class to the LogMonitoring pods. By default, no class is set|-|string|
|`resources`|Define resources' requests and limits for all the LogMonitoring pods|-|object|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode for the...|-|string|
|`tolerations`|Set tolerations for the LogMonitoring pods|-|array|

### .spec.templates.otelCollector

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the OtelCollector pods|-|object|
|`labels`|Adds additional labels for the OtelCollector pods|-|object|
|`replicas`|Number of replicas for your OtelCollector|-|integer|
|`resources`|Define resources' requests and limits for single OtelCollector pod|-|object|
|`tlsRefName`||-|string|
|`tolerations`|Set tolerations for the OtelCollector pods|-|array|
|`topologySpreadConstraints`|Adds TopologySpreadConstraints for the OtelCollector pods|-|array|

### .spec.oneAgent.classicFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer.|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.|-|boolean|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent image. Defaults to the latest image from the Dynatrace cluster.|-|string|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container.|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set.|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`storageHostPath`|StorageHostPath is the writable directory on the host filesystem where OneAgent configurations will...|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet.|-|array|
|`version`|Use a specific OneAgent version. Defaults to the latest version from the Dynatrace cluster.|-|string|

### .spec.oneAgent.cloudNativeFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent annotations.|-|object|
|`args`|Set additional arguments to the OneAgent installer.|-|array|
|`autoUpdate`|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.|-|boolean|
|`codeModulesImage`|Use a custom OneAgent CodeModule image to download binaries.|-|string|
|`dnsPolicy`|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.|-|string|
|`env`|Set additional environment variables for the OneAgent pods.|-|array|
|`image`|Use a custom OneAgent image. Defaults to the latest image from the Dynatrace cluster.|-|string|
|`initResources`|Define resources requests and limits for the initContainer.|-|object|
|`labels`|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|`namespaceSelector`|Applicable only for applicationMonitoring or cloudNativeFullStack configuration types.|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|`oneAgentResources`|Resource settings for OneAgent container.|-|object|
|`priorityClassName`|Assign a priority class to the OneAgent pods. By default, no class is set.|-|string|
|`secCompProfile`|The SecComp Profile that will be configured in order to run in secure computing mode.|-|string|
|`storageHostPath`|StorageHostPath is the writable directory on the host filesystem where OneAgent configurations will...|-|string|
|`tolerations`|Tolerations to include with the OneAgent DaemonSet.|-|array|
|`version`|Use a specific OneAgent version. Defaults to the latest version from the Dynatrace cluster.|-|string|

### .spec.oneAgent.applicationMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`codeModulesImage`|Use a custom OneAgent CodeModule image to download binaries.|-|string|
|`initResources`|Define resources requests and limits for the initContainer.|-|object|
|`namespaceSelector`|Applicable only for applicationMonitoring or cloudNativeFullStack configuration types.|-|object|
|`version`|Use a specific OneAgent CodeModule version.|-|string|

### .spec.activeGate.persistentVolumeClaim

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`accessModes`|accessModes contains the desired access modes the volume should have.<br/>More info: https://kubernetes.|-|array|
|`dataSource`|dataSource field can be used to specify either:<br/>* An existing VolumeSnapshot object (snapshot.|-|object|
|`resources`|resources represents the minimum resources the volume should have.|-|object|
|`selector`|selector is a label query over volumes to consider for binding.|-|object|
|`storageClassName`|storageClassName is the name of the StorageClass required by the claim.|-|string|
|`volumeAttributesClassName`|volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim.|-|string|
|`volumeMode`|volumeMode defines what type of volume is required by the claim.|-|string|
|`volumeName`|volumeName is the binding reference to the PersistentVolume backing this claim.|-|string|

### .spec.templates.logMonitoring.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the image to use|-|string|

### .spec.templates.otelCollector.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the image to use|-|string|

### .spec.templates.extensionExecutionController

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the ExtensionExecutionController pods|-|object|
|`customConfig`|Defines name of ConfigMap containing custom configuration file|-|string|
|`customExtensionCertificates`|Defines name of Secret containing certificates for custom extensions signature validation|-|string|
|`labels`|Adds additional labels for the ExtensionExecutionController pods|-|object|
|`resources`|Define resources' requests and limits for single ExtensionExecutionController pod|-|object|
|`tlsRefName`||-|string|
|`tolerations`|Set tolerations for the ExtensionExecutionController pods|-|array|
|`topologySpreadConstraints`|Adds TopologySpreadConstraints for the ExtensionExecutionController pods|-|array|
|`useEphemeralVolume`|Selects EmptyDir volume to be storage device|-|boolean|

### .spec.templates.kspmNodeConfigurationCollector

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations for the NodeConfigurationCollector pods|-|object|
|`args`|Set additional arguments to the NodeConfigurationCollector pods|-|array|
|`env`|Set additional environment variables for the NodeConfigurationCollector pods|-|array|
|`labels`|Adds additional labels for the NodeConfigurationCollector pods|-|object|
|`nodeSelector`|Specify the node selector that controls on which nodes NodeConfigurationCollector pods will be...|-|object|
|`priorityClassName`|If specified, indicates the pod's priority.|-|string|
|`resources`|Define resources' requests and limits for single NodeConfigurationCollector pod|-|object|
|`tolerations`|Set tolerations for the NodeConfigurationCollector pods|-|array|

### .spec.activeGate.persistentVolumeClaim.dataSourceRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiGroup`|APIGroup is the group for the resource being referenced.|-|string|
|`kind`|Kind is the type of resource being referenced|-|string|
|`name`|Name is the name of resource being referenced|-|string|
|`namespace`|Namespace is the namespace of resource being referenced<br/>Note that when a namespace is specified, a...|-|string|

### .spec.templates.extensionExecutionController.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the image to use|-|string|

### .spec.templates.kspmNodeConfigurationCollector.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the image to use|-|string|

### .spec.templates.kspmNodeConfigurationCollector.nodeAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the affinity expressions specified...|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity requirements specified by this field are not met at<br/>scheduling time, the pod will...|-|object|

### .spec.templates.kspmNodeConfigurationCollector.updateStrategy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`type`|Type of daemon set update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.|-|string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`accessModes`|accessModes contains the desired access modes the volume should have.<br/>More info: https://kubernetes.|-|array|
|`dataSource`|dataSource field can be used to specify either:<br/>* An existing VolumeSnapshot object (snapshot.|-|object|
|`resources`|resources represents the minimum resources the volume should have.|-|object|
|`selector`|selector is a label query over volumes to consider for binding.|-|object|
|`storageClassName`|storageClassName is the name of the StorageClass required by the claim.|-|string|
|`volumeAttributesClassName`|volumeAttributesClassName may be used to set the VolumeAttributesClass used by this claim.|-|string|
|`volumeMode`|volumeMode defines what type of volume is required by the claim.|-|string|
|`volumeName`|volumeName is the binding reference to the PersistentVolume backing this claim.|-|string|

### .spec.templates.kspmNodeConfigurationCollector.updateStrategy.rollingUpdate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`maxSurge`|The maximum number of nodes with an existing available DaemonSet pod that<br/>can have an updated...|-|integer or string|
|`maxUnavailable`|The maximum number of DaemonSet pods that can be unavailable during the<br/>update.|-|integer or string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim.dataSourceRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiGroup`|APIGroup is the group for the resource being referenced.|-|string|
|`kind`|Kind is the type of resource being referenced|-|string|
|`name`|Name is the name of resource being referenced|-|string|
|`namespace`|Namespace is the namespace of resource being referenced<br/>Note that when a namespace is specified, a...|-|string|
