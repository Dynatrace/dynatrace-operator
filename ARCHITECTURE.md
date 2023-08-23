# Architecture
This document describes the high-level architecture of `Dynatrace Operator`.
If you want to familiarize yourself with the code base, you are just in the right place!

## Bird's Eye View
<picture-here>

On a very high level, what the operator does is for a given `CustomResource`(CR) provided by the user, the `Operator` will deploy _one or several_ Dynatrace components into the Kubernetes Environment.

A bit more specifically:
- A `CustomResource`(CR) is configured by the user, where they provide what features or components they want to use, and provide some minimal configuration in the CR so the `Dynatrace Operator` knows what to deploy and how to configure it.
- The `Operator` not only deploys the different Dynatrace components, but also keeps them up to date.
   - The `CustomResource`(CR) defines a state, the `Dynatrace Operator` enforces it, makes it happen.

### Dynatrace Operator components

The `Dynatrace Operator` is not a single Pod, it consists of multiple components, encompassing several Kubernetes concepts.

#### Operator
This component/pod is the one that _reacts to_ our`CustomResource(s)`.
If one of our `CustomResource` is created/updated/deleted, the `Operator` is called, which cause a reconcile to happen.

A _reconcile_ just means that it will check what is in the `CR` and according to that creates/updates/deletes resources in the Kubernetes environment. (So the state of the Kubernetes Environment matches the state described in the `CR`)

#### Webhook

#### Init-Container

#### CSI-Driver


