## Setup a local Kubernetes cluster using KinD
kind/setup: prerequisites/yq
	./hack/kind/setup.sh $(K8S_VERSION)
