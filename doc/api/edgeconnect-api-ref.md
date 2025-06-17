## EdgeConnect schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional...|-|object|
|`apiServer`|Location of the...|-|string|
|`autoUpdate`|Enables automatic...|-|boolean|
|`caCertsRef`|Adds custom root...|-|string|
|`customPullSecret`|Pull secret for your...|-|string|
|`env`|Adds additional...|-|array|
|`hostPatterns`|Host patterns to be set...|-|array|
|`hostRestrictions`|Restrict outgoing HTTP...|-|array|
|`labels`|Adds additional labels...|-|object|
|`nodeSelector`|Node selector to control...|-|object|
|`replicas`|Amount of replicas for...|-|integer|
|`resources`|Defines resources...|-|object|
|`serviceAccountName`|ServiceAccountName that...|-|string|
|`tolerations`|Sets tolerations for the...|-|array|
|`topologySpreadConstraints`|Sets topology spread...|-|array|

### .spec.oauth

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`clientSecret`|Name of the secret that...|-|string|
|`endpoint`|Token endpoint URL of...|-|string|
|`provisioner`|Determines if the...|-|boolean|
|`resource`|URN identifying your...|-|string|

### .spec.proxy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`authRef`|Secret name which...|-|string|
|`host`|Server address (hostname...|-|string|
|`noProxy`|NoProxy represents the...|-|string|
|`port`|Port of the proxy.|-|integer|

### .spec.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the...|-|string|

### .spec.kubernetesAutomation

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`enabled`|Enables Kubernetes...|-|boolean|
