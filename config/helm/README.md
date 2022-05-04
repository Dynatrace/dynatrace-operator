# Tool Prerequisites

Please follow the instructions here to install mpdev and apply the application CRD:

https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/25b17f74106d4ad1e92bb1f8cfb2febc863760f8/docs/tool-prerequisites.md

# Installation

Generate an API and a PaaS token in your Dynatrace environment.

https://www.dynatrace.com/support/help/reference/dynatrace-concepts/why-do-i-need-an-environment-id/#create-user-generated-access-tokens

The Dynatrace Operator acts on its separate namespace `dynatrace`.
To create this namespace run the following command:

```
kubectl create namespace dynatrace
```

To install the Dynatrace Operator run this command after replacing apiToken and passToken:

```
mpdev /scripts/install \
--deployer=gcr.io/cloud-marketplace/dynatrace-marketplace-prod/dynatrace-operator/deployer \
--parameters='{ \
"name": "dynatrace-operator", \
"namespace": "dynatrace", \
"apiToken": "DYNATRACE_API_TOKEN", \
"paasToken": "PLATFORM_AS_A_SERVICE_TOKEN" }'
```