**Note: Requires Dynatrace Cluster version 1.209**

# Dynatrace Operator

The Dynatrace Operator supports rollout and lifecycle of various Dynatrace components in Kubernetes and OpenShift.

As of launch, the Dynatrace Operator can be used to deploy a containerized ActiveGate for Kubernetes API monitoring. New
capabilities will be added to the Dynatrace Operator over time including metric routing, and API monitoring for AWS,
Azure, GCP, and vSphere.

With v0.2.0 we added the classicFullStack functionality which allows rolling out the OneAgent to your Kubernetes
cluster. Furthermore, the Dynatrace Operator is now capable of rolling out a containerized ActiveGate for routing the
OneAgent traffic.

With v0.3.0 the CRD structure has been changed to provide an easier and more understandable way to deploy Dynatrace in your environment.
- **routing** and **kubernetesMonitoring** within the Dynakube spec are deprecated now and moved to the **activeGate** section.
- (BETA, when used with 'useCSIDriver') added **applicationMonitoring** mode, a webhook based injection mechanism for automatic-app-only injection
- added **hostMonitoring** for only monitoring the host in the cluster without app-only injection
- (BETA) added **cloudNativeFullStack** mode, which combines **hostMonitoring**, with the webhook based **applicationMonitoring**

For more information please have a look at [our DynaKube Custom Resource examples](https://github.com/Dynatrace/dynatrace-operator/tree/master/config/samples),
or our [official help page.](https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/)


## Supported platforms

Depending on the version of the Dynatrace Operator, it supports the following platforms:

| Dynatrace Operator version | Kubernetes | OpenShift Container Platform               |
| -------------------------- | ---------- | ------------------------------------------ |
| master                     | 1.20+      | 3.11.188+, 4.7+                            |
| v0.3.0                     | 1.20+      | 3.11.188+, 4.7+                            |
| v0.2.2                     | 1.18+      | 3.11.188+, 4.5+                            |
| v0.1.0                     | 1.18+      | 3.11.188+, 4.4+                            |

## Quick Start

The Dynatrace Operator acts on its separate namespace `dynatrace`. It holds the operator deployment and all dependent
objects like permissions, custom resources and corresponding StatefulSets.

#### Kubernetes

<details><summary>Installation</summary>

To create the namespace and apply the operator run the following commands

```sh
$ kubectl create namespace dynatrace
$ kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
```

A secret holding tokens for authenticating to the Dynatrace cluster needs to be created upfront. Create access tokens of
type *Dynatrace API* and *Platform as a Service* and use its values in the following commands respectively. For
assistance please refer
to [Create user-generated access tokens.](https://www.dynatrace.com/support/help/get-started/introduction/why-do-i-need-an-access-token-and-an-environment-id/#create-user-generated-access-tokens)

Make sure the *Dynatrace API* token has the following permission:

* Access problem and event feed, metrics and topology

```sh
$ kubectl -n dynatrace create secret generic dynakube --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="paasToken=PLATFORM_AS_A_SERVICE_TOKEN"
```

#### Create `DynaKube` custom resource for ActiveGate and OneAgent rollout

The rollout of the Dynatrace components is governed by a custom resource of type `DynaKube`. This custom resource will
contain parameters for various Dynatrace capabilities (API monitoring, routing, etc.)

Note: `.spec.tokens` denotes the name of the secret holding access tokens. If not specified Dynatrace Operator searches
for a secret called like the DynaKube custom resource `.metadata.name`.

```yaml
apiVersion: dynatrace.com/v1beta1
kind: DynaKube
metadata:
  name: dynakube
  namespace: dynatrace
spec:
  # Dynatrace apiUrl including the `/api` path at the end.
  # For SaaS, set `YOUR_ENVIRONMENT_ID` to your environment ID.
  # For Managed, change the apiUrl address.
  # For instructions on how to determine the environment ID and how to configure the apiUrl address, see https://www.dynatrace.com/support/help/reference/dynatrace-concepts/environment-id/.
  apiUrl: https://ENVIRONMENTID.live.dynatrace.com/api

  # name of secret holding `apiToken` and `paasToken`
  # if unset, name of custom resource is used
  #
  # tokens: ""

  # Optional: Sets Network Zone for OneAgent and ActiveGate pods
  # Make sure networkZones are enabled on your cluster before (see https://www.dynatrace.com/support/help/setup-and-configuration/network-zones/network-zones-basic-info/)
  #
  # networkZone: name-of-my-network-zone

  oneAgent:
    # enable classic fullstack monitoring and change its settings
    # Cannot be used in conjunction with cloud-native fullstack monitoring, application-only monitoring or host monitoring
    classicFullStack:

      # Optional: tolerations to include with the OneAgent DaemonSet.
      # See more here: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
      tolerations:
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists

  # Configuration for ActiveGate instances.
  activeGate:
    # Enables listed ActiveGate capabilities
    capabilities:
      - routing
      - kubernetes-monitoring

```

This is the most basic configuration for the DynaKube object. We recommend you to use classic Fullstack injection to roll out Dynatrace to your cluster, as shown in the example above.
In case you want to have adjustments please have a look at [our DynaKube Custom Resource examples](https://github.com/Dynatrace/dynatrace-operator/tree/master/config/samples).
Save one of the sample configurations, change the API url to your environment and apply it to your cluster.

```sh
$ kubectl apply -f cr.yaml
```

For detailed instructions see
our [official help page.](https://www.dynatrace.com/support/help/setup-and-configuration/setup-on-container-platforms/kubernetes/)

</details>
<details><summary>Uninstall</summary>

## Uninstall dynatrace-operator

Remove DynaKube custom resources and clean-up all remaining Dynatrace Operator specific objects:

```sh
$ kubectl delete -n dynatrace dynakube --all
$ kubectl delete -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
```

</details>

#### OpenShift

<details><summary>Installation</summary>

To create the namespace and apply the operator run the following commands (for OpenShift 4.x)

```sh
$ oc adm new-project --node-selector="" dynatrace
$ oc apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/openshift.yaml
```

If you are using *OpenShift 3.11*, make sure to run the following commands, instead of the ones above

```sh
$ oc adm new-project --node-selector="" dynatrace
$ oc apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/openshift3.11.yaml
```

A secret holding tokens for authenticating to the Dynatrace cluster needs to be created upfront. Create access tokens of
type *Dynatrace API* and *Platform as a Service* and use its values in the following commands respectively. For
assistance please refere
to [Create user-generated access tokens.](https://www.dynatrace.com/support/help/get-started/introduction/why-do-i-need-an-access-token-and-an-environment-id/#create-user-generated-access-tokens)

Make sure the *Dynatrace API* token has the following permission:

* Access problem and event feed, metrics and topology

```sh
$ oc -n dynatrace create secret generic dynakube --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="paasToken=PLATFORM_AS_A_SERVICE_TOKEN"
```

#### Create `DynaKube` custom resource for ActiveGate and OneAgent rollout

The rollout of the Dynatrace components is governed by a custom resource of type `DynaKube`. This custom resource will
contain parameters for various Dynatrace capabilities (API monitoring, routing, etc.)

Note: `.spec.tokens` denotes the name of the secret holding access tokens. If not specified Dynatrace Operator searches
for a secret called like the DynaKube custom resource `.metadata.name`.

```yaml
apiVersion: dynatrace.com/v1beta1
kind: DynaKube
metadata:
  name: dynakube
  namespace: dynatrace
spec:
  # Dynatrace apiUrl including the `/api` path at the end.
  # For SaaS, set `YOUR_ENVIRONMENT_ID` to your environment ID.
  # For Managed, change the apiUrl address.
  # For instructions on how to determine the environment ID and how to configure the apiUrl address, see https://www.dynatrace.com/support/help/reference/dynatrace-concepts/environment-id/.
  apiUrl: https://ENVIRONMENTID.live.dynatrace.com/api

  # name of secret holding `apiToken` and `paasToken`
  # if unset, name of custom resource is used
  #
  # tokens: ""

  # Optional: Sets Network Zone for OneAgent and ActiveGate pods
  # Make sure networkZones are enabled on your cluster before (see https://www.dynatrace.com/support/help/setup-and-configuration/network-zones/network-zones-basic-info/)
  #
  # networkZone: name-of-my-network-zone

  oneAgent:
    # enable classic fullstack monitoring and change its settings
    # Cannot be used in conjunction with cloud-native fullstack monitoring, application-only monitoring or host monitoring
    classicFullStack:

      # Optional: tolerations to include with the OneAgent DaemonSet.
      # See more here: https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
      tolerations:
        - effect: NoSchedule
          key: node-role.kubernetes.io/master
          operator: Exists

  # Configuration for ActiveGate instances.
  activeGate:
    # Enables listed ActiveGate capabilities
    capabilities:
      - routing
      - kubernetes-monitoring
      - data-ingest

```

This is the most basic configuration for the DynaKube object. We recommend you to use classic Fullstack injection to roll out Dynatrace to your cluster, as shown in the example above.
In case you want to have adjustments please have a look at [our DynaKube Custom Resource examples](https://github.com/Dynatrace/dynatrace-operator/tree/master/config/samples).
Save one of the sample configurations, change the API url to your environment and apply it to your cluster.

```sh
$ oc apply -f cr.yaml
```

For detailed instructions see
our [official help page.](https://www.dynatrace.com/support/help/technology-support/cloud-platforms/openshift/monitor-openshift-environments/)

</details>
<details><summary>Uninstall</summary>

## Uninstall dynatrace-operator

Remove DynaKube custom resources and clean-up all remaining Dynatrace Operator specific objects:

```sh
$ oc delete -n dynatrace dynakube --all
$ oc delete -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/openshift.yaml
```

</details>

## Hacking

See [HACKING](HACKING.md) for details on how to get started enhancing Dynatrace Operator.

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details on submitting changes.

## License

Dynatrace Operator is under Apache 2.0 license. See [LICENSE](LICENSE) for details.
