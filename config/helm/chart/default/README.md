# Dynatrace Operator Helm Chart

The Dynatrace Operator supports rollout and lifecycle of various Dynatrace components in Kubernetes and OpenShift.

This Helm Chart requires Helm 3.

## Quick Start

Migration instructions can be found in the [official help page](https://www.dynatrace.com/support/help/shortlink/k8s-dto-helm#migrate).

Install the Dynatrace Operator via Helm by running the following commands.

### Installation

> For instructions on how to install the dynatrace-operator on Openshift, head to the
> [official help page](https://www.dynatrace.com/support/help/shortlink/kubernetes)

#### For versions older than 0.15.0

Add `dynatrace` helm repository:

```console
helm repo add dynatrace https://raw.githubusercontent.com/Dynatrace/dynatrace-operator/main/config/helm/repos/stable
```

Install `dynatrace-operator` helm chart and create the corresponding `dynatrace` namespace:

```console
helm install dynatrace-operator dynatrace/dynatrace-operator -n dynatrace --create-namespace --atomic
```

#### For versions 0.15.0 and after

Install `dynatrace-operator` helm chart using the OCI repository and create the corresponding `dynatrace` namespace:

```console
helm install dynatrace-operator oci://public.ecr.aws/dynatrace/dynatrace-operator  -n dynatrace --create-namespace --atomic
```

## Selecting the operator image

By default, the chart uses the operator image of the platform (for example `public.ecr.aws/dynatrace/dynatrace-operator`) with the tag matching the chart version.

Use `imageRef` to change that:

- `imageRef.tag`: overrides the default tag, the platform's default repository is kept. The tag is used as is, no `v` prefix is added.
- `imageRef.repository`: overrides the default repository.

For example, to install a FIPS image from the default repository:

```console
helm install dynatrace-operator oci://public.ecr.aws/dynatrace/dynatrace-operator -n dynatrace --create-namespace --atomic \
  --set imageRef.tag=<version>-fips
```

The obsolete `image` value takes precedence over everything in `imageRef`.

## Uninstall chart

> Full instructions can be found in the [official help page](https://www.dynatrace.com/support/help/shortlink/guides-k8s-update-uninstall-operator)

Uninstall the Dynatrace Operator by running the following command:

```console
helm uninstall dynatrace-operator -n dynatrace
```
