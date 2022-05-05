# Tool Prerequisites

Please follow the instructions here to install mpdev and apply the application CRD:

https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/25b17f74106d4ad1e92bb1f8cfb2febc863760f8/docs/tool-prerequisites.md

# Installation

The Dynatrace Operator acts on its separate namespace `dynatrace`.
To create this namespace run the following command:

```
kubectl create namespace dynatrace
```

To install the Dynatrace Operator run this command:

```
mpdev /scripts/install \
--deployer=gcr.io/cloud-marketplace/dynatrace-marketplace-prod/dynatrace-operator/deployer \
--parameters='{ \
"name": "dynatrace-operator", \
"namespace": "dynatrace" }'
```