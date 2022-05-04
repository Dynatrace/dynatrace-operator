# Hacking

The `upload_test.sh` script will help you test your local changes.
It grabs your current selected gcloud project, an application name and a version, builds a new deployer image with all the corresponding files in your repository and pushes it.
Afterwards it applies the application CRD (from Google) and uses mpdev (needs to be part of $PATH) to deploy that new deployer image to your current Kubernetes cluster.

The following environment variables needs to be provided:
* gcloud project (REGISTRY bases on this)
* APP_NAME (defaults to "dynatrace-operator")
* VERSION (tag of the docker image - defaults to "test")
* APITOKEN (dynatrace api token)
* PAASTOKEN (dynatrace paas token)

The `verify.sh` script will run will build/push the necessary container then run `mpdev verify` on it.
