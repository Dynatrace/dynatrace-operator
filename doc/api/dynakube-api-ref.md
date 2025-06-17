## DynaKube schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiUrl`|Dynatrace apiUrl,...|-|string|
|`customPullSecret`|Defines a custom pull...|-|string|
|`dynatraceApiRequestThreshold`|Configuration for...|-|integer|
|`enableIstio`|When enabled, and if...|-|boolean|
|`extensions`|When an (empty)...|-|object|
|`networkZone`|Sets a network zone for...|-|string|
|`proxy`|Set custom proxy...|-|object|
|`skipCertCheck`|Disable certificate...|-|boolean|
|`tokens`|Name of the secret...|-|string|
|`trustedCAs`|Adds custom RootCAs from...|-|string|

### .spec.kspm

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`mappedHostPaths`|MappedHostPaths define...|-|array|

### .spec.oneAgent

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`hostGroup`|Sets a host group for...|-|string|

### .spec.activeGate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional...|-|object|
|`capabilities`|Activegate capabilities...|-|array|
|`customProperties`|Add a custom properties...|-|object|
|`dnsPolicy`|Sets DNS Policy for the...|-|string|
|`env`|List of environment...|-|array|
|`group`|Set activation group for...|-|string|
|`image`|The ActiveGate container...|-|string|
|`labels`|Adds additional labels...|-|object|
|`nodeSelector`|Node selector to control...|-|object|
|`priorityClassName`|If specified, indicates...|-|string|
|`replicas`|Amount of replicas for...|-|integer|
|`resources`|Define resources...|-|object|
|`terminationGracePeriodSeconds`|Configures the...|-|integer|
|`tlsSecretName`|The name of a secret...|-|string|
|`tolerations`|Set tolerations for the...|-|array|
|`topologySpreadConstraints`|Adds...|-|array|
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
|`enabled`|Enables...|-|boolean|
|`namespaceSelector`|The namespaces where you...|-|object|

### .spec.oneAgent.hostMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent...|-|object|
|`args`|Set additional arguments...|-|array|
|`autoUpdate`|Disables automatic...|-|boolean|
|`dnsPolicy`|Set the DNS Policy for...|-|string|
|`env`|Set additional...|-|array|
|`image`|Use a custom OneAgent...|-|string|
|`labels`|Your defined labels for...|-|object|
|`nodeSelector`|Specify the node...|-|object|
|`oneAgentResources`|Resource settings for...|-|object|
|`priorityClassName`|Assign a priority class...|-|string|
|`secCompProfile`|The SecComp Profile that...|-|string|
|`storageHostPath`|StorageHostPath is the...|-|string|
|`tolerations`|Tolerations to include...|-|array|
|`version`|Use a specific OneAgent...|-|string|

### .spec.templates.logMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom annotations...|-|object|
|`args`|Set additional arguments...|-|array|
|`dnsPolicy`|Sets DNS Policy for the...|-|string|
|`labels`|Add custom labels to the...|-|object|
|`nodeSelector`|Node selector to control...|-|object|
|`priorityClassName`|Assign a priority class...|-|string|
|`resources`|Define resources'...|-|object|
|`secCompProfile`|The SecComp Profile that...|-|string|
|`tolerations`|Set tolerations for the...|-|array|

### .spec.templates.otelCollector

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional...|-|object|
|`labels`|Adds additional labels...|-|object|
|`replicas`|Number of replicas for...|-|integer|
|`resources`|Define resources'...|-|object|
|`tlsRefName`||-|string|
|`tolerations`|Set tolerations for the...|-|array|
|`topologySpreadConstraints`|Adds...|-|array|

### .spec.oneAgent.classicFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent...|-|object|
|`args`|Set additional arguments...|-|array|
|`autoUpdate`|Disables automatic...|-|boolean|
|`dnsPolicy`|Set the DNS Policy for...|-|string|
|`env`|Set additional...|-|array|
|`image`|Use a custom OneAgent...|-|string|
|`labels`|Your defined labels for...|-|object|
|`nodeSelector`|Specify the node...|-|object|
|`oneAgentResources`|Resource settings for...|-|object|
|`priorityClassName`|Assign a priority class...|-|string|
|`secCompProfile`|The SecComp Profile that...|-|string|
|`storageHostPath`|StorageHostPath is the...|-|string|
|`tolerations`|Tolerations to include...|-|array|
|`version`|Use a specific OneAgent...|-|string|

### .spec.oneAgent.cloudNativeFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Add custom OneAgent...|-|object|
|`args`|Set additional arguments...|-|array|
|`autoUpdate`|Disables automatic...|-|boolean|
|`codeModulesImage`|Use a custom OneAgent...|-|string|
|`dnsPolicy`|Set the DNS Policy for...|-|string|
|`env`|Set additional...|-|array|
|`image`|Use a custom OneAgent...|-|string|
|`initResources`|Define resources...|-|object|
|`labels`|Your defined labels for...|-|object|
|`namespaceSelector`|Applicable only for...|-|object|
|`nodeSelector`|Specify the node...|-|object|
|`oneAgentResources`|Resource settings for...|-|object|
|`priorityClassName`|Assign a priority class...|-|string|
|`secCompProfile`|The SecComp Profile that...|-|string|
|`storageHostPath`|StorageHostPath is the...|-|string|
|`tolerations`|Tolerations to include...|-|array|
|`version`|Use a specific OneAgent...|-|string|

### .spec.activeGate.volumeClaimTemplate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`accessModes`|accessModes contains the...|-|array|
|`dataSource`|dataSource field can be...|-|object|
|`resources`|resources represents the...|-|object|
|`selector`|selector is a label...|-|object|
|`storageClassName`|storageClassName is the...|-|string|
|`volumeAttributesClassName`|volumeAttributesClassName...|-|string|
|`volumeMode`|volumeMode defines what...|-|string|
|`volumeName`|volumeName is the...|-|string|

### .spec.oneAgent.applicationMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`codeModulesImage`|Use a custom OneAgent...|-|string|
|`initResources`|Define resources...|-|object|
|`namespaceSelector`|Applicable only for...|-|object|
|`version`|Use a specific OneAgent...|-|string|

### .spec.templates.logMonitoring.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the...|-|string|

### .spec.templates.otelCollector.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the...|-|string|

### .spec.templates.extensionExecutionController

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional...|-|object|
|`customConfig`|Defines name of...|-|string|
|`customExtensionCertificates`|Defines name of Secret...|-|string|
|`labels`|Adds additional labels...|-|object|
|`resources`|Define resources'...|-|object|
|`tlsRefName`||-|string|
|`tolerations`|Set tolerations for the...|-|array|
|`topologySpreadConstraints`|Adds...|-|array|
|`useEphemeralVolume`|Selects EmptyDir volume...|-|boolean|

### .spec.templates.kspmNodeConfigurationCollector

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional...|-|object|
|`args`|Set additional arguments...|-|array|
|`env`|Set additional...|-|array|
|`labels`|Adds additional labels...|-|object|
|`nodeSelector`|Specify the node...|-|object|
|`priorityClassName`|If specified, indicates...|-|string|
|`resources`|Define resources'...|-|object|
|`tolerations`|Set tolerations for the...|-|array|

### .spec.activeGate.volumeClaimTemplate.dataSourceRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiGroup`|APIGroup is the group...|-|string|
|`kind`|Kind is the type of...|-|string|
|`name`|Name is the name of...|-|string|
|`namespace`|Namespace is the...|-|string|

### .spec.templates.extensionExecutionController.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the...|-|string|

### .spec.templates.kspmNodeConfigurationCollector.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the...|-|string|

### .spec.templates.kspmNodeConfigurationCollector.nodeAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will...|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity...|-|object|

### .spec.templates.kspmNodeConfigurationCollector.updateStrategy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`type`|Type of daemon set...|-|string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`accessModes`|accessModes contains the...|-|array|
|`dataSource`|dataSource field can be...|-|object|
|`resources`|resources represents the...|-|object|
|`selector`|selector is a label...|-|object|
|`storageClassName`|storageClassName is the...|-|string|
|`volumeAttributesClassName`|volumeAttributesClassName...|-|string|
|`volumeMode`|volumeMode defines what...|-|string|
|`volumeName`|volumeName is the...|-|string|

### .spec.templates.kspmNodeConfigurationCollector.updateStrategy.rollingUpdate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`maxSurge`|The maximum number of...|-|integer or string|
|`maxUnavailable`|The maximum number of...|-|integer or string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim.dataSourceRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiGroup`|APIGroup is the group...|-|string|
|`kind`|Kind is the type of...|-|string|
|`name`|Name is the name of...|-|string|
|`namespace`|Namespace is the...|-|string|
