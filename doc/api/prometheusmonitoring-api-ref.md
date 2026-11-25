## PrometheusMonitoring schema

### .spec

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`dynaKubeName`|Name of the DynaKube in the same namespace that provides all connection<br/>settings (apiUrl, tokens, proxy, networkZone, trustedCAs, ActiveGate, etc.).|-|string|
|`publicRegistryOverride`|Overrides the default registry from which Dynatrace images are pulled.|-|string|

### .spec.gateway

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Extra annotations merged into the pod template metadata.|-|object|
|`args`|Command-line arguments for the main application container.|-|array|
|`env`|Additional environment variables for the main application container.<br/>Entries are appended to the ones set by the operator; if a name is used twice, the last entry (the one defined here) wins.|-|array|
|`image`|Overrides the default component image (full reference including tag or digest).<br/>When omitted the operator uses the latest available image from the public Dynatrace registry.|-|string|
|`imagePullPolicy`|Image pull policy for the component pods.|-|string|
|`labels`|Extra labels merged into the pod template metadata.|-|object|
|`nodeSelector`|Node selector to control the selection of nodes for the component pods.|-|object|
|`priorityClassName`|PriorityClass assigned to the component pods so they are not evicted under<br/>node pressure.|-|string|
|`replicas`|Number of replicas for the component. At least 2 is recommended for<br/>failure tolerance.|-|integer|
|`resources`|Resource requests and limits for each component pod.|-|object|
|`tolerations`|Tolerations for the component pods.|-|array|
|`topologySpreadConstraints`|Topology spread constraints for the component pods.|-|array|
|`volumeMounts`|Additional volume mounts for the main application container.<br/>Entries are appended to the ones set by the operator, their mount paths must not collide with the operator's mounts.|-|array|
|`volumes`|Additional volumes for the component pods.<br/>Entries are appended to the ones set by the operator, their names must not collide with the operator's volumes.|-|array|

### .spec.scraper

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Extra annotations merged into the pod template metadata.|-|object|
|`args`|Command-line arguments for the main application container.|-|array|
|`env`|Additional environment variables for the main application container.<br/>Entries are appended to the ones set by the operator; if a name is used twice, the last entry (the one defined here) wins.|-|array|
|`image`|Overrides the default component image (full reference including tag or digest).<br/>When omitted the operator uses the latest available image from the public Dynatrace registry.|-|string|
|`imagePullPolicy`|Image pull policy for the component pods.|-|string|
|`labels`|Extra labels merged into the pod template metadata.|-|object|
|`nodeSelector`|Node selector to control the selection of nodes for the component pods.|-|object|
|`priorityClassName`|PriorityClass assigned to the component pods so they are not evicted under<br/>node pressure.|-|string|
|`replicas`|Number of replicas for the component. At least 2 is recommended for<br/>failure tolerance.|-|integer|
|`resources`|Resource requests and limits for each component pod.|-|object|
|`targetsPollInterval`|Default interval to poll target allocator for scrape targets.|60s|string|
|`tolerations`|Tolerations for the component pods.|-|array|
|`topologySpreadConstraints`|Topology spread constraints for the component pods.|-|array|
|`volumeMounts`|Additional volume mounts for the main application container.<br/>Entries are appended to the ones set by the operator, their mount paths must not collide with the operator's mounts.|-|array|
|`volumes`|Additional volumes for the component pods.<br/>Entries are appended to the ones set by the operator, their names must not collide with the operator's volumes.|-|array|

### .spec.targetAllocator

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`annotations`|Extra annotations merged into the pod template metadata.|-|object|
|`args`|Command-line arguments for the main application container.|-|array|
|`customResourceNamespaceSelector`|Restricts which namespaces the TA watches for CRDs. An empty selector {}<br/>matches all namespaces.|-|object|
|`customResourceSelector`|Label selector applied to all Prometheus Operator CRDs (ServiceMonitor,<br/>PodMonitor, ScrapeConfig, Probe). The TA only picks up CRDs whose labels<br/>match. When omitted, the operator defaults to matching<br/>prometheus.dynatrace.com: "true". An empty selector {} matches all CRDs.|{'matchLabels': {'prometheus.dynatrace.com': 'true'}}|object|
|`env`|Additional environment variables for the main application container.<br/>Entries are appended to the ones set by the operator; if a name is used twice, the last entry (the one defined here) wins.|-|array|
|`image`|Overrides the default component image (full reference including tag or digest).<br/>When omitted the operator uses the latest available image from the public Dynatrace registry.|-|string|
|`imagePullPolicy`|Image pull policy for the component pods.|-|string|
|`labels`|Extra labels merged into the pod template metadata.|-|object|
|`nodeSelector`|Node selector to control the selection of nodes for the component pods.|-|object|
|`priorityClassName`|PriorityClass assigned to the component pods so they are not evicted under<br/>node pressure.|-|string|
|`replicas`|Number of replicas for the component. At least 2 is recommended for<br/>failure tolerance.|-|integer|
|`resources`|Resource requests and limits for each component pod.|-|object|
|`scrapeInterval`|Default interval used by the target allocator to look for Prometheus CRs<br/>(ServiceMonitor, PodMonitor, ScrapeConfig, Probe).|60s|string|
|`tolerations`|Tolerations for the component pods.|-|array|
|`topologySpreadConstraints`|Topology spread constraints for the component pods.|-|array|
|`volumeMounts`|Additional volume mounts for the main application container.<br/>Entries are appended to the ones set by the operator, their mount paths must not collide with the operator's mounts.|-|array|
|`volumes`|Additional volumes for the component pods.<br/>Entries are appended to the ones set by the operator, their names must not collide with the operator's volumes.|-|array|

### .spec.scraper.strategy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`type`|Type of deployment. Can be "Recreate" or "RollingUpdate". Default is RollingUpdate.|-|string|

### .spec.gateway.updateStrategy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`type`|Type indicates the type of the StatefulSetUpdateStrategy.<br/>Default is RollingUpdate.|-|string|

### .spec.targetAllocator.strategy

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`type`|Type of deployment. Can be "Recreate" or "RollingUpdate". Default is RollingUpdate.|-|string|

### .spec.gateway.affinity.podAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and adding<br/>"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to a pod label update), the<br/>system may or may not try to eventually evict the pod from its node.<br/>When there are multiple elements, the lists of nodes corresponding to each<br/>podAffinityTerm are intersected, i.e. all terms must be satisfied.|-|array|

### .spec.scraper.affinity.podAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and adding<br/>"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to a pod label update), the<br/>system may or may not try to eventually evict the pod from its node.<br/>When there are multiple elements, the lists of nodes corresponding to each<br/>podAffinityTerm are intersected, i.e. all terms must be satisfied.|-|array|

### .spec.gateway.affinity.nodeAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and adding<br/>"weight" to the sum if the node matches the corresponding matchExpressions; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to an update), the system<br/>may or may not try to eventually evict the pod from its node.|-|object|

### .spec.scraper.affinity.nodeAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and adding<br/>"weight" to the sum if the node matches the corresponding matchExpressions; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to an update), the system<br/>may or may not try to eventually evict the pod from its node.|-|object|

### .spec.scraper.strategy.rollingUpdate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`maxSurge`|The maximum number of pods that can be scheduled above the desired number of<br/>pods.<br/>Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).<br/>This can not be 0 if MaxUnavailable is 0.<br/>Absolute number is calculated from percentage by rounding up.<br/>Defaults to 25%.<br/>Example: when this is set to 30%, the new ReplicaSet can be scaled up immediately when<br/>the rolling update starts, such that the total number of old and new pods do not exceed<br/>130% of desired pods. Once old pods have been killed,<br/>new ReplicaSet can be scaled up further, ensuring that total number of pods running<br/>at any time during the update is at most 130% of desired pods.|-|integer or string|
|`maxUnavailable`|The maximum number of pods that can be unavailable during the update.<br/>Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).<br/>Absolute number is calculated from percentage by rounding down.<br/>This can not be 0 if MaxSurge is 0.<br/>Defaults to 25%.<br/>Example: when this is set to 30%, the old ReplicaSet can be scaled down to 70% of desired pods<br/>immediately when the rolling update starts. Once new pods are ready, old ReplicaSet<br/>can be scaled down further, followed by scaling up the new ReplicaSet, ensuring<br/>that the total number of pods available at all times during the update is at<br/>least 70% of desired pods.|-|integer or string|

### .spec.gateway.affinity.podAntiAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the anti-affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling anti-affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and subtracting<br/>"weight" from the sum if the node has pods which matches the corresponding podAffinityTerm; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the anti-affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the anti-affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to a pod label update), the<br/>system may or may not try to eventually evict the pod from its node.<br/>When there are multiple elements, the lists of nodes corresponding to each<br/>podAffinityTerm are intersected, i.e. all terms must be satisfied.|-|array|

### .spec.scraper.affinity.podAntiAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the anti-affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling anti-affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and subtracting<br/>"weight" from the sum if the node has pods which matches the corresponding podAffinityTerm; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the anti-affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the anti-affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to a pod label update), the<br/>system may or may not try to eventually evict the pod from its node.<br/>When there are multiple elements, the lists of nodes corresponding to each<br/>podAffinityTerm are intersected, i.e. all terms must be satisfied.|-|array|

### .spec.gateway.updateStrategy.rollingUpdate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`maxUnavailable`|The maximum number of pods that can be unavailable during the update.<br/>Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).<br/>Absolute number is calculated from percentage by rounding up. This can not be 0.<br/>Defaults to 1. This field is beta-level and is enabled by default. The field applies to all pods in the range 0 to<br/>Replicas-1. That means if there is any unavailable pod in the range 0 to Replicas-1, it<br/>will be counted towards MaxUnavailable.<br/>This setting might not be effective for the OrderedReady podManagementPolicy. That policy ensures pods are created and become ready one at a time.|-|integer or string|
|`partition`|Partition indicates the ordinal at which the StatefulSet should be partitioned<br/>for updates. During a rolling update, all pods from ordinal Replicas-1 to<br/>Partition are updated. All pods from ordinal Partition-1 to 0 remain untouched.<br/>This is helpful in being able to do a canary based deployment. The default value is 0.|-|integer|

### .spec.targetAllocator.affinity.podAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and adding<br/>"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to a pod label update), the<br/>system may or may not try to eventually evict the pod from its node.<br/>When there are multiple elements, the lists of nodes corresponding to each<br/>podAffinityTerm are intersected, i.e. all terms must be satisfied.|-|array|

### .spec.targetAllocator.affinity.nodeAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and adding<br/>"weight" to the sum if the node matches the corresponding matchExpressions; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to an update), the system<br/>may or may not try to eventually evict the pod from its node.|-|object|

### .spec.targetAllocator.strategy.rollingUpdate

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`maxSurge`|The maximum number of pods that can be scheduled above the desired number of<br/>pods.<br/>Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).<br/>This can not be 0 if MaxUnavailable is 0.<br/>Absolute number is calculated from percentage by rounding up.<br/>Defaults to 25%.<br/>Example: when this is set to 30%, the new ReplicaSet can be scaled up immediately when<br/>the rolling update starts, such that the total number of old and new pods do not exceed<br/>130% of desired pods. Once old pods have been killed,<br/>new ReplicaSet can be scaled up further, ensuring that total number of pods running<br/>at any time during the update is at most 130% of desired pods.|-|integer or string|
|`maxUnavailable`|The maximum number of pods that can be unavailable during the update.<br/>Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).<br/>Absolute number is calculated from percentage by rounding down.<br/>This can not be 0 if MaxSurge is 0.<br/>Defaults to 25%.<br/>Example: when this is set to 30%, the old ReplicaSet can be scaled down to 70% of desired pods<br/>immediately when the rolling update starts. Once new pods are ready, old ReplicaSet<br/>can be scaled down further, followed by scaling up the new ReplicaSet, ensuring<br/>that the total number of pods available at all times during the update is at<br/>least 70% of desired pods.|-|integer or string|

### .spec.targetAllocator.affinity.podAntiAffinity

|Parameter|Description|Default value|Data type|
|:-|:-|:-|:-|
|`preferredDuringSchedulingIgnoredDuringExecution`|The scheduler will prefer to schedule pods to nodes that satisfy<br/>the anti-affinity expressions specified by this field, but it may choose<br/>a node that violates one or more of the expressions. The node that is<br/>most preferred is the one with the greatest sum of weights, i.e.<br/>for each node that meets all of the scheduling requirements (resource<br/>request, requiredDuringScheduling anti-affinity expressions, etc.),<br/>compute a sum by iterating through the elements of this field and subtracting<br/>"weight" from the sum if the node has pods which matches the corresponding podAffinityTerm; the<br/>node(s) with the highest sum are the most preferred.|-|array|
|`requiredDuringSchedulingIgnoredDuringExecution`|If the anti-affinity requirements specified by this field are not met at<br/>scheduling time, the pod will not be scheduled onto the node.<br/>If the anti-affinity requirements specified by this field cease to be met<br/>at some point during pod execution (e.g. due to a pod label update), the<br/>system may or may not try to eventually evict the pod from its node.<br/>When there are multiple elements, the lists of nodes corresponding to each<br/>podAffinityTerm are intersected, i.e. all terms must be satisfied.|-|array|
