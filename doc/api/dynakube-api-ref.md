## DynaKube schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|apiUrl|Dynatrace apiUrl, including the /api path at the end. For SaaS, set YOUR_ENVIRONMENT_ID to your environment ID. For Managed, change the apiUrl address. For instructions on how to determine the environment ID and how to configure the apiUrl address, see Environment ID (<https://www.dynatrace.com/support/help/get-started/monitoring-environment/environment-id>).|-|string|
|customPullSecret|Defines a custom pull secret in case you use a private registry when pulling images from the Dynatrace environment. To define a custom pull secret and learn about the expected behavior, see Configure customPullSecret (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#custompullsecret>).|-|string|
|enableIstio|When enabled, and if Istio is installed on the Kubernetes environment, Dynatrace Operator will create the corresponding VirtualService and ServiceEntry objects to allow access to the Dynatrace Cluster from the OneAgent or ActiveGate. Disabled by default.|-|boolean|
|namespaceSelector|Applicable only for applicationMonitoring or cloudNativeFullStack configuration types. The namespaces where you want Dynatrace Operator to inject. For more information, see Configure monitoring for namespaces and pods (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring/dto-config-options-k8s#annotate>).|-|object|
|networkZone|Sets a network zone for the OneAgent and ActiveGate pods.|-|string|
|proxy|Set custom proxy settings either directly or from a secret with the field proxy. Note: Applies to Dynatrace Operator, ActiveGate, and OneAgents.|-|object|
|skipCertCheck|Disable certificate check for the connection between Dynatrace Operator and the Dynatrace Cluster. Set to true if you want to skip certification validation checks.|-|boolean|
|tokens|Name of the secret holding the tokens used for connecting to Dynatrace.|-|string|
|trustedCAs|Adds custom RootCAs from a configmap. Put the certificate under certs within your configmap. Note: Applies only to Dynatrace Operator and OneAgent, not to ActiveGate.|-|string|

### .spec.activeGate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|annotations|Adds additional annotations to the ActiveGate pods|-|object|
|capabilities|Activegate capabilities enabled (routing, kubernetes-monitoring, metrics-ingest, dynatrace-api)|-|array|
|customProperties|Add a custom properties file by providing it as a value or reference it from a secret If referenced from a secret, make sure the key is called 'customProperties'|-|object|
|dnsPolicy|Sets DNS Policy for the ActiveGate pods|-|string|
|env|List of environment variables to set for the ActiveGate|-|array|
|group|Set activation group for ActiveGate|-|string|
|image|The ActiveGate container image. Defaults to the latest ActiveGate image provided by the registry on the tenant|-|string|
|labels|Adds additional labels for the ActiveGate pods|-|object|
|nodeSelector|Node selector to control the selection of nodes|-|object|
|priorityClassName|If specified, indicates the pod's priority. Name must be defined by creating a PriorityClass object with that name. If not specified the setting will be removed from the StatefulSet.|-|string|
|replicas|Amount of replicas for your ActiveGates|-|integer|
|resources|Define resources requests and limits for single ActiveGate pods|-|object|
|tlsSecretName|The name of a secret containing ActiveGate TLS cert+key and password. If not set, self-signed certificate is used. server.p12: certificate+key pair in pkcs12 format password: passphrase to read server.p12|-|string|
|tolerations|Set tolerations for the ActiveGate pods|-|array|
|topologySpreadConstraints|Adds TopologySpreadConstraints for the ActiveGate pods|-|array|

### .spec.oneAgent.hostMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|annotations|Add custom OneAgent annotations.|-|object|
|args|Set additional arguments to the OneAgent installer. For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>). For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|autoUpdate|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>). Enabled by default.|-|boolean|
|dnsPolicy|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|env|Set additional environment variables for the OneAgent pods.|-|array|
|image|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|labels|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|nodeSelector|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|oneAgentResources|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR. Note: resource.requests shows the values needed to run; resource.limits shows the maximum limits for the pod.|-|object|
|priorityClassName|Assign a priority class to the OneAgent pods. By default, no class is set. For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|tolerations|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|version|The OneAgent version to be used.|-|string|

### .spec.oneAgent.classicFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|annotations|Add custom OneAgent annotations.|-|object|
|args|Set additional arguments to the OneAgent installer. For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>). For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|autoUpdate|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>). Enabled by default.|-|boolean|
|dnsPolicy|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|env|Set additional environment variables for the OneAgent pods.|-|array|
|image|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|labels|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|nodeSelector|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|oneAgentResources|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR. Note: resource.requests shows the values needed to run; resource.limits shows the maximum limits for the pod.|-|object|
|priorityClassName|Assign a priority class to the OneAgent pods. By default, no class is set. For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|tolerations|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|version|The OneAgent version to be used.|-|string|

### .spec.oneAgent.cloudNativeFullStack

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|annotations|Add custom OneAgent annotations.|-|object|
|args|Set additional arguments to the OneAgent installer. For available options, see Linux custom installation (<https://www.dynatrace.com/support/help/setup-and-configuration/dynatrace-oneagent/installation-and-operation/linux/installation/customize-oneagent-installation-on-linux>). For the list of limitations, see Limitations (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/docker/set-up-dynatrace-oneagent-as-docker-container#limitations>).|-|array|
|autoUpdate|Disables automatic restarts of OneAgent pods in case a new version is available (<https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/get-started-with-kubernetes-monitoring#disable-auto>). Enabled by default.|-|boolean|
|codeModulesImage|The OneAgent image that is used to inject into Pods.|-|string|
|dnsPolicy|Set the DNS Policy for OneAgent pods. For details, see Pods DNS Policy (<https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy>).|-|string|
|env|Set additional environment variables for the OneAgent pods.|-|array|
|image|Use a custom OneAgent Docker image. Defaults to the image from the Dynatrace cluster.|-|string|
|initResources|Define resources requests and limits for the initContainer. For details, see Managing resources for containers (<https://kubernetes.io/docs/concepts/configuration/manage-resources-containers>).|-|object|
|labels|Your defined labels for OneAgent pods in order to structure workloads as desired.|-|object|
|nodeSelector|Specify the node selector that controls on which nodes OneAgent will be deployed.|-|object|
|oneAgentResources|Resource settings for OneAgent container. Consumption of the OneAgent heavily depends on the workload to monitor. You can use the default settings in the CR. Note: resource.requests shows the values needed to run; resource.limits shows the maximum limits for the pod.|-|object|
|priorityClassName|Assign a priority class to the OneAgent pods. By default, no class is set. For details, see Pod Priority and Preemption (<https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/>).|-|string|
|tolerations|Tolerations to include with the OneAgent DaemonSet. For details, see Taints and Tolerations (<https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/>).|-|array|
|version|The OneAgent version to be used.|-|string|

### .spec.oneAgent.applicationMonitoring

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|codeModulesImage|The OneAgent image that is used to inject into Pods.|-|string|
|initResources|Define resources requests and limits for the initContainer. For details, see Managing resources for containers (<https://kubernetes.io/docs/concepts/configuration/manage-resources-containers>).|-|object|
|useCSIDriver|Set if you want to use the CSIDriver. Don't enable it if you do not have access to Kubernetes nodes or if you lack privileges.|-|boolean|
|version|The OneAgent version to be used.|-|string|
