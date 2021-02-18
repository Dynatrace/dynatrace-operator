**Note: Requires Dynatrace Cluster version 1.209**

# Dynatrace Operator

The Dynatrace Operator supports rollout and lifecycle of various Dynatrace components in Kubernetes and OpenShift.

As of launch, the Dynatrace Operator can be used to deploy a containerized ActiveGate for Kubernetes API monitoring. New capabilities will be added to the Dynatrace Operator over time including metric routing, and API monitoring for AWS, Azure, GCP, and vSphere.

## Supported platforms

Depending on the version of the Dynatrace Operator, it supports the following platforms:

| Dynatrace Operator version | Kubernetes | OpenShift Container Platform               |
| -------------------------- | ---------- | ------------------------------------------ |
| master                     | 1.18+      | 3.11.188+, 4.4+                            |
| v0.1.0                     | 1.18+      | 3.11.188+, 4.4+                            |

## Quick Start

The Dynatrace Operator acts on its separate namespace `dynatrace`. It holds the operator deployment and all dependent objects like permissions, custom resources and corresponding StatefulSets.

#### Kubernetes

<details><summary>Installation</summary>

To create the namespace and apply the operator run the following commands

```sh
$ kubectl create namespace dynatrace
$ kubectl apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/kubernetes.yaml
```

A secret holding tokens for authenticating to the Dynatrace cluster needs to be created upfront.
Create access tokens of type *Dynatrace API* and *Platform as a Service* and use its values in the following commands respectively.
For assistance please refere to [Create user-generated access tokens](https://www.dynatrace.com/support/help/get-started/introduction/why-do-i-need-an-access-token-and-an-environment-id/#create-user-generated-access-tokens).
```sh
$ kubectl -n dynatrace create secret generic dynakube --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="paasToken=PLATFORM_AS_A_SERVICE_TOKEN"
```

#### Create `DynaKube` custom resource for ActiveGate rollout

The rollout of Dynatrace ActiveGate is governed by a custom resource of type `DynaKube`. This custom resource will contain parameters for various Dynatrace capabilities (API monitoring, routing, etc.)

Note: `.spec.tokens` denotes the name of the secret holding access tokens. If not specified Dynatrace Operator searches for a secret called like the DynaKube custom resource `.metadata.name`.
```yaml
apiVersion: dynatrace.com/v1alpha1
kind: DynaKube
metadata:
  name: dynakube
  namespace: dynatrace
spec:
  # dynatrace api url including `/api` path at the end
  # either set ENVIRONMENTID to the proper tenant id or change the apiUrl as a whole, e.q. for Managed
  #
  apiUrl: https://ENVIRONMENTID.live.dynatrace.com/api

  # name of secret holding `apiToken` and `paasToken`
  # if unset, name of custom resource is used
  #
  # tokens: ""

  # Enables and configures an ActiveGate instance that allows monitoring
  # of Kubernetes environments
  #
  kubernetesMonitoring:
    #   Enable Kubernetes monitoring functionality
    #
    enabled: true
```
This is the most basic configuration for the DynaKube object. In case you want to have adjustments please have a look at [our DynaKube Custom Resource example](https://raw.githubusercontent.com/Dynatrace/dynatrace-operator/master/config/samples/cr.yaml). 
Save this to cr.yaml and apply it to your cluster.
```sh
$ kubectl apply -f cr.yaml
```
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

To create the namespace and apply the operator run the following commands

```sh
$ oc adm new-project --node-selector="" dynatrace
$ oc apply -f https://github.com/Dynatrace/dynatrace-operator/releases/latest/download/openshift.yaml
```

A secret holding tokens for authenticating to the Dynatrace cluster needs to be created upfront.
Create access tokens of type *Dynatrace API* and *Platform as a Service* and use its values in the following commands respectively.
For assistance please refere to [Create user-generated access tokens](https://www.dynatrace.com/support/help/get-started/introduction/why-do-i-need-an-access-token-and-an-environment-id/#create-user-generated-access-tokens).
```sh
$ oc -n dynatrace create secret generic dynakube --from-literal="apiToken=DYNATRACE_API_TOKEN" --from-literal="paasToken=PLATFORM_AS_A_SERVICE_TOKEN"
```

#### Create `DynaKube` custom resource for ActiveGate rollout

The rollout of Dynatrace ActiveGate is governed by a custom resource of type `DynaKube`.

Note: `.spec.tokens` denotes the name of the secret holding access tokens. If not specified Dynatrace Operator searches for a secret called like the DynaKube custom resource `.metadata.name`.
```yaml
apiVersion: dynatrace.com/v1alpha1
kind: DynaKube
metadata:
  name: dynakube
  namespace: dynatrace
spec:
  # dynatrace api url including `/api` path at the end
  # either set ENVIRONMENTID to the proper tenant id or change the apiUrl as a whole, e.q. for Managed
  #
  apiUrl: https://ENVIRONMENTID.live.dynatrace.com/api

  # name of secret holding `apiToken` and `paasToken`
  # if unset, name of custom resource is used
  #
  # tokens: ""

  # Enables and configures an ActiveGate instance that allows monitoring
  # of Kubernetes environments
  #
  kubernetesMonitoring:
    #   Enable Kubernetes monitoring functionality
    #
    enabled: true
```
This is the most basic configuration for the DynaKube object. In case you want to have adjustments please have a look at [our DynaKube Custom Resource example](https://raw.githubusercontent.com/Dynatrace/dynatrace-operator/master/config/samples/cr.yaml). 
Save this to cr.yaml and apply it to your cluster.
```sh
$ oc apply -f cr.yaml
```
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
