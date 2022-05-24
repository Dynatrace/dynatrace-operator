-include ../prerequisites.mk
-include ../images.mk
-include ../manifests/*.mk

## Deploy the operator in the Kubernetes cluster configured in ~/.kube/config
deploy/kubernetes: manifests/kubernetes prerequisites/kustomize
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	cd config/deploy/kubernetes && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(BRANCH_IMAGE)
	$(KUSTOMIZE) build config/deploy/kubernetes | kubectl apply -f -
