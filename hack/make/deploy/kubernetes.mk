## Deploy the operator in the Kubernetes cluster configured in ~/.kube/config
deploy/kubernetes: manifests/kubernetes prerequisites/kustomize
	kubectl get namespace dynatrace || kubectl create namespace dynatrace
	cd $(MANIFESTS_DIR)/kubernetes && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(BRANCH_IMAGE)
	$(KUSTOMIZE) build $(MANIFESTS_DIR)/kubernetes | kubectl apply -f -
