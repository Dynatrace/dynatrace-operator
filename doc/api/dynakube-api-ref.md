## DynaKube schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiUrl`||-|string|
|`customPullSecret`||-|string|
|`dynatraceApiRequestThreshold`||-|integer|
|`enableIstio`||-|boolean|
|`extensions`||-|object|
|`kspm`||-|object|
|`networkZone`||-|string|
|`proxy`||-|object|
|`skipCertCheck`||-|boolean|
|`tokens`||-|string|
|`trustedCAs`||-|string|

### .spec.oneAgent

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`hostGroup`||-|string|

### .spec.activeGate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`capabilities`||-|array|
|`customProperties`||-|object|
|`dnsPolicy`||-|string|
|`env`||-|array|
|`group`||-|string|
|`image`||-|string|
|`labels`||-|object|
|`nodeSelector`||-|object|
|`priorityClassName`||-|string|
|`replicas`||-|integer|
|`resources`||-|object|
|`terminationGracePeriodSeconds`||-|integer|
|`tlsSecretName`||-|string|
|`tolerations`||-|array|
|`topologySpreadConstraints`||-|array|
|`useEphemeralVolume`||-|boolean|

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
|`enabled`||-|boolean|
|`namespaceSelector`||-|object|

### .spec.oneAgent.hostMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`args`||-|array|
|`autoUpdate`||-|boolean|
|`dnsPolicy`||-|string|
|`env`||-|array|
|`image`||-|string|
|`labels`||-|object|
|`nodeSelector`||-|object|
|`oneAgentResources`||-|object|
|`priorityClassName`||-|string|
|`secCompProfile`||-|string|
|`storageHostPath`||-|string|
|`tolerations`||-|array|
|`version`||-|string|

### .spec.templates.logMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`args`||-|array|
|`dnsPolicy`||-|string|
|`labels`||-|object|
|`nodeSelector`||-|object|
|`priorityClassName`||-|string|
|`resources`||-|object|
|`secCompProfile`||-|string|
|`tolerations`||-|array|

### .spec.templates.otelCollector

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`labels`||-|object|
|`replicas`||-|integer|
|`resources`||-|object|
|`tlsRefName`||-|string|
|`tolerations`||-|array|
|`topologySpreadConstraints`||-|array|

### .spec.oneAgent.classicFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`args`||-|array|
|`autoUpdate`||-|boolean|
|`dnsPolicy`||-|string|
|`env`||-|array|
|`image`||-|string|
|`labels`||-|object|
|`nodeSelector`||-|object|
|`oneAgentResources`||-|object|
|`priorityClassName`||-|string|
|`secCompProfile`||-|string|
|`storageHostPath`||-|string|
|`tolerations`||-|array|
|`version`||-|string|

### .spec.oneAgent.cloudNativeFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`args`||-|array|
|`autoUpdate`||-|boolean|
|`codeModulesImage`||-|string|
|`dnsPolicy`||-|string|
|`env`||-|array|
|`image`||-|string|
|`initResources`||-|object|
|`labels`||-|object|
|`namespaceSelector`||-|object|
|`nodeSelector`||-|object|
|`oneAgentResources`||-|object|
|`priorityClassName`||-|string|
|`secCompProfile`||-|string|
|`storageHostPath`||-|string|
|`tolerations`||-|array|
|`version`||-|string|

### .spec.oneAgent.applicationMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`codeModulesImage`||-|string|
|`initResources`||-|object|
|`namespaceSelector`||-|object|
|`version`||-|string|

### .spec.activeGate.persistentVolumeClaim

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`accessModes`||-|array|
|`dataSource`||-|object|
|`resources`||-|object|
|`selector`||-|object|
|`storageClassName`||-|string|
|`volumeAttributesClassName`||-|string|
|`volumeMode`||-|string|
|`volumeName`||-|string|

### .spec.templates.logMonitoring.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`||-|string|
|`tag`||-|string|

### .spec.templates.otelCollector.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`||-|string|
|`tag`||-|string|

### .spec.templates.extensionExecutionController

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`customConfig`||-|string|
|`customExtensionCertificates`||-|string|
|`labels`||-|object|
|`resources`||-|object|
|`tlsRefName`||-|string|
|`tolerations`||-|array|
|`topologySpreadConstraints`||-|array|
|`useEphemeralVolume`||-|boolean|

### .spec.templates.kspmNodeConfigurationCollector

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`args`||-|array|
|`env`||-|array|
|`labels`||-|object|
|`nodeSelector`||-|object|
|`priorityClassName`||-|string|
|`resources`||-|object|
|`tolerations`||-|array|

### .spec.activeGate.persistentVolumeClaim.dataSourceRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiGroup`||-|string|
|`kind`||-|string|
|`name`||-|string|
|`namespace`||-|string|

### .spec.templates.extensionExecutionController.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`||-|string|
|`tag`||-|string|

### .spec.templates.kspmNodeConfigurationCollector.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`||-|string|
|`tag`||-|string|

### .spec.templates.kspmNodeConfigurationCollector.nodeAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`||-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`||-|object|

### .spec.templates.kspmNodeConfigurationCollector.updateStrategy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`type`||-|string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`accessModes`||-|array|
|`dataSource`||-|object|
|`resources`||-|object|
|`selector`||-|object|
|`storageClassName`||-|string|
|`volumeAttributesClassName`||-|string|
|`volumeMode`||-|string|
|`volumeName`||-|string|

### .spec.templates.kspmNodeConfigurationCollector.updateStrategy.rollingUpdate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`maxSurge`||-|integer or string|
|`maxUnavailable`||-|integer or string|

### .spec.templates.extensionExecutionController.persistentVolumeClaim.dataSourceRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`apiGroup`||-|string|
|`kind`||-|string|
|`name`||-|string|
|`namespace`||-|string|
