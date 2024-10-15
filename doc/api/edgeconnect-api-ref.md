## EdgeConnect schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Adds additional annotations to the EdgeConnect pods|-|object|
|`apiServer`|Location of the Dynatrace API to connect to, including your specific environment UUID|-|string|
|`autoUpdate`|Enables automatic restarts of EdgeConnect pods in case a new version is available (the default value is: true)|True|boolean|
|`caCertsRef`|Adds custom root certificate from a configmap. Put the certificate under certs within your configmap.|-|string|
|`customPullSecret`|Pull secret for your private registry|-|string|
|`env`|Adds additional environment variables to the EdgeConnect pods|-|array|
|`hostPatterns`|Host patterns to be set in the tenant, only considered when provisioning is enabled.|-|array|
|`hostRestrictions`|Restrict outgoing HTTP requests to your internal resources to specified hosts|-|array|
|`labels`|Adds additional labels to the EdgeConnect pods|-|object|
|`nodeSelector`|Node selector to control the selection of nodes for the EdgeConnect pods|-|object|
|`replicas`|Amount of replicas for your EdgeConnect (the default value is: 1)|1|integer|
|`resources`|Defines resources requests and limits for single pods|-|object|
|`serviceAccountName`|ServiceAccountName that allows EdgeConnect to access the Kubernetes API|dynatrace-edgeconnect|string|
|`tolerations`|Sets tolerations for the EdgeConnect pods|-|array|
|`topologySpreadConstraints`|Sets topology spread constraints for the EdgeConnect pods|-|array|

### .spec.oauth

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`clientSecret`|Name of the secret that holds oauth clientId/secret|-|string|
|`endpoint`|Token endpoint URL of Dynatrace SSO|-|string|
|`provisioner`|Determines if the operator will create the EdgeConnect and light OAuth client on the cluster using the credentials provided. Requires more scopes than default behavior.|-|boolean|
|`resource`|URN identifying your account. You get the URN when creating the OAuth client|-|string|

### .spec.proxy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`authRef`|Secret name which contains the username and password used for authentication with the proxy, using the<br/>"Basic" HTTP authentication scheme.|-|string|
|`host`|Server address (hostname or IP address) of the proxy.|-|string|
|`noProxy`|NoProxy represents the NO_PROXY or no_proxy environment<br/>variable. It specifies a string that contains comma-separated values<br/>specifying hosts that should be excluded from proxying.|-|string|
|`port`|Port of the proxy.|-|integer|

### .spec.imageRef

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`repository`|Custom image repository|-|string|
|`tag`|Indicates a tag of the image to use|-|string|

### .spec.kubernetesAutomation

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`enabled`|Enables Kubernetes Automation for Workflows|-|boolean|
