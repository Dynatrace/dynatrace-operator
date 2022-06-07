# Dynatrace Operator Helm Chart

The Dynatrace Operator supports rollout and lifecycle of various Dynatrace components in Kubernetes and OpenShift.

This Helm Chart requires Helm 3.

## Platforms
Depending on the version the platforms shown in the [Supported platforms](../../../../README.md##Supported%20platforms) section are supported.

## Quick Start


To install the Dynatrace Operator via Helm run the following commands.

### Installation
Add `dynatrace` helm repository:
```
helm repo add dynatrace https://raw.githubusercontent.com/Dynatrace/dynatrace-operator/master/config/helm/repos/stable
```

> For instructions on how to uninstall the dynatrace-operator on Openshift, head to the [official help page](https://www.dynatrace.com/support/help/shortlink/k8s-helm)

Install `dynatrace-operator` helm chart and create the corresponding `dynatrace` namespace:
```console
helm install dynatrace-operator dynatrace/dynatrace-operator -n dynatrace --create-namespace --atomic
```

## Uninstall chart

```console
helm uninstall dynatrace-operator -n dynatrace
```
