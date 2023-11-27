EdgeConnect is the Schema for the EdgeConnect API
## EdgeConnect is the Schema for the EdgeConnect API
### .spec
|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|annotations|Adds additional annotations to the EdgeConnect pods|-|object|
|apiServer|Location of the Dynatrace API to connect to, including your specific environment UUID|-|string|
|autoUpdate|Enables automatic restarts of EdgeConnect pods in case a new version is available (the default value is: true)|True|boolean|
|customPullSecret|Pull secret for your private registry|-|string|
|env|Adds additional environment variables to the EdgeConnect pods|-|array|
|hostRestrictions|Restrict outgoing HTTP requests to your internal resources to specified hosts|-|string|
|labels|Adds additional labels to the EdgeConnect pods|-|object|
|nodeSelector|Node selector to control the selection of nodes for the EdgeConnect pods|-|object|
|replicas|Amount of replicas for your EdgeConnect (the default value is: 1)|1|integer|
|resources|Defines resources requests and limits for single pods|-|object|
|tolerations|Sets tolerations for the EdgeConnect pods|-|array|
|topologySpreadConstraints|Sets topology spread constraints for the EdgeConnect pods|-|array|

### .spec.oauth
|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|clientSecret|Name of the secret that holds oauth clientId/secret|-|string|
|endpoint|Token endpoint URL of Dynatrace SSO|-|string|
|resource|URN identifying your account. You get the URN when creating the OAuth client|-|string|

### .spec.imageRef
|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|repository|Custom EdgeConnect image repository|-|string|
|tag|Indicates version of the EdgeConnect image to use|-|string|

