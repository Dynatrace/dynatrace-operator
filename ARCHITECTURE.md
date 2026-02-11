# Architecture

This document describes the high-level architecture of `Dynatrace Operator`.
If you want to familiarize yourself with the code base, you are just in the right place!

- [Architecture](#architecture)
  - [Bird's Eye View](#birds-eye-view)
  - [Custom Resources](#custom-resources)
    - [DynaKube](#dynakube)
    - [EdgeConnect](#edgeconnect)
  - [Dynatrace Operator Components](#dynatrace-operator-components)
    - [Main Operator Pod](#main-operator-pod)
    - [Webhook Pod](#webhook-pod)
    - [Bootstrapper (Init Container)](#bootstrapper-init-container)
    - [CSI Driver](#csi-driver)
    - [Generate-metadata command](#generate-metadata-command)
    - [Support Tools](#support-tools)
  - [Codebase Structure](#codebase-structure)
    - [Key Design Patterns](#key-design-patterns)
  - [Development Workflow](#development-workflow)
    - [Binary Modes](#binary-modes)
    - [Reconciliation Flow](#reconciliation-flow)
    - [Testing Strategy](#testing-strategy)
  - [Additional Resources](#additional-resources)

## Bird's Eye View

```mermaid
graph LR
    A[fa:fa-user User] -->|creates| B(fa:fa-file CR)
    subgraph kubernetes
    B --> |triggers| C(fa:fa-wrench Operator)
    C -->|deploys| D[Dynatrace-Component-1]:::dk1
    C -->|deploys| E[Dynatrace-Component-2]:::dk2
    C -->|deploys| F[Dynatrace-Component-3]:::dk3
    end

    classDef dk1 stroke:#f00
    classDef dk2 stroke:#0f0
    classDef dk3 stroke:#00f
```

On a very high level, what the operator does is for a given `CustomResource`(CR) provided by the user, the `Operator` will deploy _one or several_ Dynatrace components into the Kubernetes Environment.

A bit more specifically:

- A `CustomResource`(CR) is configured by the user, where they provide what features or components they want to use, and provide some minimal configuration in the CR so the `Dynatrace Operator` knows what to deploy and how to configure it.
- The `Operator` not only deploys the different Dynatrace components, but also keeps them up to date.
  - The `CustomResource`(CR) defines a state, the `Dynatrace Operator` enforces it, makes it happen.

## Custom Resources

The Dynatrace Operator supports two main Custom Resource Definitions (CRDs):

### DynaKube

The primary CRD for deploying and managing Dynatrace observability components. The latest API version is stored in `pkg/api/latest/dynakube/`, with versioned APIs maintained for [backward compatibility](https://docs.dynatrace.com/docs/ingest-from/setup-on-k8s/guides/migration/dynakube#deprecation) (at the time of writing: v1beta3, v1beta4 and v1beta5).

**Key Features:**

- **OneAgent deployment, supported modes:**
  - `classicFullStack`: Pod per node for full-stack monitoring
  - `applicationMonitoring`: Webhook-based app-only injection with optional CSI driver caching
  - `hostMonitoring`: Node-only monitoring with [optional CSI driver](https://docs.dynatrace.com/docs/shortlink/how-it-works-k8s-operator#csidriver) for read-only operation
  - `cloudNativeFullStack`: Combined application and host monitoring
- **ActiveGate deployment, supported capabilities:**
  - `routing`: Routes OneAgent traffic through an [ActiveGate](https://docs.dynatrace.com/docs/ingest-from/dynatrace-activegate)
  - `kubernetes-monitoring`: Monitors Kubernetes API
  - `metrics-ingest`: Routes enriched metrics through ActiveGate
- **Additional feature deployments:**
  - [Extensions](https://docs.dynatrace.com/docs/ingest-from/extensions)
    - Disclaimer: Only a limited number of extensions are supported by the operator.
  - Log monitoring
  - OpenTelemetry Collector
  - [KSPM](https://www.dynatrace.com/news/blog/kubernetes-security-posture-management-kspm/) (Kubernetes Security Posture Management)
  - Metadata enrichment

### EdgeConnect

Manages [Dynatrace EdgeConnect](https://docs.dynatrace.com/docs/ingest-from/edgeconnect) deployments for extending observability to remote locations. The latest API version is `v1alpha2` stored in [pkg/api/v1alpha2/edgeconnect](pkg/api/v1alpha2/edgeconnect).

## Dynatrace Operator Components

The `Dynatrace Operator` is not a single Pod, it consists of multiple components working together, utilizing several Kubernetes concepts.

### Main Operator Pod

The central controller [`cmd/operator/`](./cmd/operator/) that reconciles Custom Resources. It consists of multiple sub-reconcilers:

**DynaKube Controller** [`pkg/controllers/dynakube/`](./pkg/controllers/dynakube/):

The DynaKube is a rather large CR, therefore its controller has many feature-specific sub-reconcilers each with nested components. Here are the top-level ones:

- [`activegate`](./pkg/controllers/dynakube/activegate/): Manages ActiveGate StatefulSets
- [`oneagent`](./pkg/controllers/dynakube/oneagent/): Handles [OneAgent](https://docs.dynatrace.com/docs/ingest-from/dynatrace-oneagent) DaemonSets for host monitoring
- [`injection`](./pkg/controllers/dynakube/injection/): Manages code module / [OTLP](https://opentelemetry.io/docs/specs/otlp/) / metadata enrichment injection into application pods
- [`extension`](./pkg/controllers/dynakube/extension/): Controls Dynatrace extensions deployment
- [`otelc`](./pkg/controllers/dynakube/otelc/): Manages OpenTelemetry Collector deployment
- [`logmonitoring`](./pkg/controllers/dynakube/logmonitoring/): Handles log monitoring components
- [`kspm`](./pkg/controllers/dynakube/kspm/): Manages Kubernetes Security Posture Management
- [`apimonitoring`](./pkg/controllers/dynakube/apimonitoring/): Monitors Kubernetes API
- [`istio`](./pkg/controllers/dynakube/istio/): Handles Istio service mesh integration
- [`proxy`](./pkg/controllers/dynakube/proxy/): Manages proxy configurations
- [`deploymentmetadata`](./pkg/controllers/dynakube/deploymentmetadata/): Manages deployment metadata. This is a small amount of metadata about the Operator and the OneAgent deployment mode, provided to the OneAgents.
  - should be moved to the [`oneagent`](./pkg/controllers/dynakube/oneagent/) folder in the future

> [!WARNING]
> This is not the best pattern, it is the case mainly due to historical reasons, we will try to improve this in the future.

**EdgeConnect Controller** [(`pkg/controllers/edgeconnect/`)](./pkg/controllers/edgeconnect/):

- Manages EdgeConnect deployments.

**Node Controller** [(`pkg/controllers/nodes/`)](./pkg/controllers/nodes/):

- Monitors node lifecycle and maintains node-level state. Used for notifying the Dynatrace Environment if a node goes down in an expected way. So the users will not see false positives in the Dynatrace UI.
- Its future is uncertain, we will try to remove it in the future.

**Certificates Controller** [(`pkg/controllers/certificates/`)](./pkg/controllers/certificates/):

- Creates self-signed TLS certificates for our (mutating/validating/conversion) Webhooks. Really old, meant to make the install seamless for the user, and not require any additional dependencies (like [cert-manager](https://cert-manager.io/) for example).
- The certs are created by the Operator pod and read by the webhook pod. Not purely handled by the webhook, as we don't want to have leader election for the webhook.

Relevant links:

- [Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

### Webhook Pod

The webhook server [(`cmd/webhook/`)](./cmd/webhook/) intercepts creation/update of Kubernetes Resources and either mutates or validates them.

**Validation Webhooks**[(`pkg/api/validation/`)](./pkg/api/validation/):

- Validates DynaKube and EdgeConnect CRs to catch misconfigurations before they're applied
  - Normally, each API version of a CR has its own validation webhook, but we only have one webhook for all API versions. This is because of the high number of API versions we have, and we don't want to duplicate the code for each API version, as that would just make the codebase more complex without any real benefit.
  - We solve this by calling the conversion logic in the validation webhook as well, so we can always validate the latest API version. This is not the most performant solution, but it's the simplest one.
- Prevents invalid changes from reaching the cluster

**Mutation Webhooks**[(`pkg/webhook/mutation/`)](./pkg/webhook/mutation/):

- **Pod Mutation** [(`pkg/webhook/mutation/pod/`)](./pkg/webhook/mutation/pod/): Injects init containers, volumes, environment variables, and annotations into user pods for application monitoring
- **Namespace Mutation** [(`pkg/webhook/mutation/namespace/`)](./pkg/webhook/mutation/namespace/): Labels namespaces to track/control which namespace should the Pod mutation webhook react to
  - May be removed in the future, as we plan to move to a more fine-grained approach.

The webhook uses TLS for secure communication and includes health/readiness probes for reliability.

Relevant links:

- [What are webhooks?](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks)

### Bootstrapper (Init Container)

The bootstrapper [`cmd/bootstrapper/`](./cmd/bootstrapper/) runs as an [init container](https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) injected into user pods via the webhook.

It can operate in three modes:

1. CSI-backed: Uses pre-downloaded [code modules](https://docs.dynatrace.com/docs/shortlink/oneagent-support-matrix#ent-cloud-code-modules) from the CSI driver
2. Direct download: Fetches [code modules](https://docs.dynatrace.com/docs/shortlink/oneagent-support-matrix#ent-cloud-code-modules) directly from Dynatrace API
3. Metadata enrichment only: Only enriches the pod with metadata

After downloading the code modules, it configures the OneAgent for the specific application and sets up metadata enrichment.

### CSI Driver

A [Container Storage Interface](https://github.com/container-storage-interface/spec/blob/master/spec.md) driver [`cmd/csi/`](./cmd/csi/) that provides volumes for OneAgent [code modules](https://docs.dynatrace.com/docs/shortlink/oneagent-support-matrix#ent-cloud-code-modules).

The CSI driver optimizes disk space usage by sharing OneAgent binaries across multiple pods on the same node and improves startup performance by caching downloads.

It consists of multiple components:

**CSI Server** [`cmd/csi/server/`](./cmd/csi/server/):

- Main CSI driver implementation running on each node
- Handles volume provisioning and mounting

It can provide 2 types of volumes:

1. [`app` volumes](./pkg/controllers/csi/server/volumes/app): These volumes contain a single OneAgent code module, and are used for application monitoring. Uses overlayfs to minimize disk space usage.
2. [`host` volumes](./pkg/controllers/csi/server/volumes/host): These volumes are just an empty directory on the node, and are used by the host OneAgents to persist their data.

**CSI Provisioner** [`cmd/csi/provisioner/`](./cmd/csi/provisioner/):

- Downloads the OneAgent code modules from the Dynatrace API and stores them on the host, to be used by the `server` container to provide volumes to the pods.
  - It can download the code modules in 3 different ways:
    - As a ZIP, from the Dynatrace Environments API, which it has to extract and move to the correct location.
    - As a tar, from an OCI Image, which it has to extract and move to the correct location.
      - This most likely will be removed in the future, in favor of the Job based approach.
    - By scheduling a [Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/), which uses an OCI image that is a self extracting code module.
- Manages the state of the filesystem, cleans up unused code modules.

**CSI Init** [`cmd/csi/init/`](./cmd/csi/init/):

- Initializes the CSI driver environment. Handles possible migrations from previous versions.

**Node Driver Registrar** [`cmd/csi/registrar/`](./cmd/csi/registrar/):

- Registers the CSI driver with the Kubelet
- Has been reimplemented by us, instead of using the [upstream implementation](https://github.com/kubernetes-csi/node-driver-registrar/tree/master), due to go version inconsistencies causing complications when handling CVE related questions.

**Liveness Probe** [`cmd/csi/livenessprobe/`](./cmd/csi/livenessprobe/):

- Monitors CSI driver health
- Has been reimplemented by us, instead of using the [upstream implementation](https://github.com/kubernetes-csi/livenessprobe/tree/master), due to go version inconsistencies causing complications when handling CVE related questions.

Relevant links:

- [CSI volume](https://kubernetes.io/docs/concepts/storage/volumes/#csi)

### Generate-metadata command

This command [`cmd/metadata/`](./cmd/metadata/) generates metadata files containing Kubernetes attributes (namespace, pod name, labels, etc.) for enriching host OneAgents.

### Support Tools

**Support Archive** [`cmd/supportarchive/`](./cmd/supportarchive/):

- Collects diagnostic information from the cluster
- Gathers operator logs, DynaKube/EdgeConnect status, and resource states
- Helps troubleshoot issues with Dynatrace support

**Troubleshoot** [`cmd/troubleshoot/`](./cmd/troubleshoot/):

> [!NOTE]
> This tool is outdated. Use the support archive command instead.

- Command-line tool for diagnosing common deployment issues
- Checks CRDs, namespaces, images, proxies, and configurations

**Startup Probe** [`cmd/startupprobe/`](./cmd/startupprobe/):

- Validates that OneAgent has started correctly in pods

## Codebase Structure

**`cmd/`** - Entry points for all executables

- Each subdirectory contains a CLI command that can be invoked
- The main binary includes all commands as subcommands via [Cobra](https://pkg.go.dev/github.com/spf13/cobra)
- Examples: `operator`, `webhook`, `csi-server`, `bootstrap`, `troubleshoot`

[**`pkg/api/`**](./pkg/api/): Custom Resource Definitions and API types

- [**`latest/`**](./pkg/api/latest/): Current API version
  - the purpose of this "hack" is to make the codebase easier to maintain, so when we introduce a new API version, we don't have to update the imports for every single file.
- [**`v1alpha1/`**](./pkg/api/v1alpha1/), [**`v1alpha2/`**](./pkg/api/v1alpha2/), [**`v1beta3/`**](./pkg/api/v1beta3/), etc. - Versioned APIs
- [**`conversion/`**](./pkg/api/conversion/): API version conversion logic
- [**`validation/`**](./pkg/api/validation/): CR validation logic
- [**`scheme/`**](./pkg/api/scheme/): Kubernetes scheme registration

[**`pkg/controllers/`**](./pkg/controllers/): Reconciliation logic

[**`pkg/webhook/`**](./pkg/webhook/): Admission webhook handlers

- [**`mutation/`**](./pkg/webhook/mutation/): Mutating webhooks for pods and namespaces

[**`pkg/clients/`**](./pkg/clients/): External API clients

- [**`dynatrace/`**](./pkg/clients/dynatrace/): Dynatrace API client
- [**`edgeconnect/`**](./pkg/clients/edgeconnect/): EdgeConnect API client

[**`pkg/injection/`**](./pkg/injection/): [Code module](https://docs.dynatrace.com/docs/shortlink/oneagent-support-matrix#ent-cloud-code-modules) injection logic

- [**`codemodule/`**](./pkg/injection/codemodule/): Code module installer and management
- [**`namespace/`**](./pkg/injection/namespace/): Namespace injection mapper

[**`pkg/util/`**](./pkg/util/): Utility packages

- Common utilities for Kubernetes operations, hashing, tokens, conditions, etc.

[**`pkg/otelcgen/`**](./pkg/otelcgen/): OpenTelemetry Collector generation

- Logic for generating OpenTelemetry Collector configurations and components

[**`pkg/logd/`**](./pkg/logd/): Logging

- Logging configuration and utilities

[**`pkg/util/oci/`**](pkg/util/oci/): OCI Image Handling

- Utilities for interacting with OCI registries and images

[**`pkg/arch/`**](./pkg/arch/): Architecture Constants

- CPU architecture specific constants and utilities

### Key Design Patterns

**Builder Pattern**: Used extensively for creating reconcilers and clients, allowing flexible configuration and testability

**Reconciler Pattern**: Each feature has its own reconciler that implements a `Reconcile()` method, composed together in the main controller

**Status Subresource**: CRs maintain a status field tracking deployment state, versions and conditions

**Watch & Reconcile**: Controllers watch for changes to CRs and owned resources, triggering reconciliation with smart backoff intervals

## Development Workflow

### Binary Modes

The main binary (`cmd/main.go`) is a multi-mode executable that behaves differently based on the subcommand:

```bash
dynatrace-operator operator                  # Run the main operator
dynatrace-operator webhook-server            # Run the webhook server
dynatrace-operator csi-server                # Run CSI driver server
dynatrace-operator csi-init                  # Run CSI driver initialization
dynatrace-operator csi-provisioner           # Run CSI driver provisioner
dynatrace-operator csi-node-driver-registrar # Run CSI node driver registrar
dynatrace-operator livenessprobe             # Run CSI liveness probe
dynatrace-operator bootstrap                 # Run bootstrapper (init container)
dynatrace-operator troubleshoot              # Run troubleshooting tool
dynatrace-operator support-archive           # Generate support bundle
dynatrace-operator startup-probe             # Run startup probe
dynatrace-operator generate-metadata         # Generate metadata file
```

This design allows using a single container image with different entry points for different components.

### Reconciliation Flow

1. **Watch**: Controller watches DynaKube/EdgeConnect CRs and owned resources
2. **Queue**: Changes trigger reconcile requests added to a work queue
3. **Reconcile**: Controller processes the request:
   - Fetches the current CR state
   - Calls sub-reconcilers for each feature
   - Updates Kubernetes resources (StatefulSets, DaemonSets, etc.)
   - Updates CR status with results
4. **Requeue**: Returns with a requeue interval (1m/5m/30m based on state)

### Testing Strategy

- **Unit Tests**: Test individual functions and components in isolation
- **Integration Tests**: Test controller behavior with fake Kubernetes clients
- **E2E Tests**: Full end-to-end testing in real clusters ([test/e2e/scenarios](./test/e2e/scenarios))
- **Mocks**: Generated using mockery for external dependencies

## Additional Resources

- [HACKING.md](HACKING.md) - Development setup and guidelines
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [Official Documentation](https://www.dynatrace.com/support/help/shortlink/kubernetes) - User-facing documentation
- [API Samples](assets/samples/dynakube/) - Example DynaKube configurations
