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
This component/pod is the one that _reacts to_ the creation/update/delete of our`CustomResource(s)`, causing the `Operator` to _reconcile_.
A _reconcile_ just means that it will check what is in the `CustomResource(s)` and according to that creates/updates/deletes resources in the Kubernetes environment. (So the state of the Kubernetes Environment matches the state described in the `CR`)

#### Webhook
This component/pod is the one that _intercepts_ creation/update/delete of Kubernetes Resources (only those that are relevant), then either mutates or validates them.
- Validation: We only use it for our `CustomResource(s)`, it's meant to catch known misconfigurations. If the validation webhook detects a problem, the user is warned, the change is denied and rolled back, like nothing happened.
- Mutation: Used to modify Kubernetes Resources "in flight", so the Resource will be created/updated in the cluster like if it was applied with added modifications.
   - We have 2 use-cases for this:
      - Seamlessly modifying user resources with the necessary configuration needed for Dynatrace observability features to work.
      - Handle time/timing sensitive minor modifications (labeling, annotating) of user resources, which is meant to help the `Operator` perform more reliably and timely.

#### Init-Container
Some configurations need to happen on the container filesystem level, like setting up a volume or creating/updating configuration files.
To achieve this we add our init-container (using the `Webhook`) to user Pods. As init-containers run before any other container, we can setup the environment of user containers to enable Dynatrace observability features.

#### CSI-Driver
A component that is present on all nodes, meant to provide volumes (based on the node's filesystem) to make the capabilities provided by the `Operator` to use less disk space and be more performant.

