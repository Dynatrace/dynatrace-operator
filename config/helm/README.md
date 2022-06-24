# Tool Prerequisites

* Install mpdev, see [google documentation](https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/docs/tool-prerequisites.md) for more information
* Create an empty GKE cluster
* Apply Googles Application CRD, see [google documentation](https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools/blob/master/docs/tool-prerequisites.md) for more information

# Installation

* Run `hack/gcr/deployer-image.sh` to build and push a new deployer image containing the helm charts and builds the operator image and pushes to the gcr registry
* Run `hack/gcr/deploy.sh` to deploy the deployer image
* In the Google Cloud Console, go to Kubernetes Clusters / Applications, select your cluster and check if the deployment was successful

# Verify

* Run `hack/gcr/verify.sh` does a combination of the `hack/gcr/deployer-image.sh` and `hack/gcr/deploy.sh` but uses `verify` instead of install
