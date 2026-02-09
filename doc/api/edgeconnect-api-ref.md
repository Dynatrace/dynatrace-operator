## EdgeConnect schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`||-|object|
|`apiServer`||-|string|
|`autoUpdate`||-|boolean|
|`caCertsRef`||-|string|
|`customPullSecret`||-|string|
|`env`||-|array|
|`hostPatterns`||-|array|
|`hostRestrictions`||-|array|
|`labels`||-|object|
|`nodeSelector`||-|object|
|`replicas`||-|integer|
|`resources`||-|object|
|`serviceAccountName`||-|string|
|`tolerations`||-|array|
|`topologySpreadConstraints`||-|array|

### .spec.oauth

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`clientSecret`||-|string|
|`endpoint`||-|string|
|`provisioner`||-|boolean|
|`resource`||-|string|

### .spec.proxy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`authRef`||-|string|
|`host`||-|string|
|`noProxy`||-|string|
|`port`||-|integer|

### .spec.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`pullPolicy`||-|string|
|`repository`||-|string|
|`tag`||-|string|

### .spec.kubernetesAutomation

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`enabled`||-|boolean|
