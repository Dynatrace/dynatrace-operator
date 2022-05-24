-include ../prerequisites.mk
-include ../images.mk
-include ../manifests/*.mk

## Deploy the operator in the OpenShift cluster configured in ~/.kube/config
deploy/openshift: manifests/openshift prerequisites/kustomize
	oc get project dynatrace || oc adm new-project --node-selector="" dynatrace
	cd config/deploy/openshift && $(KUSTOMIZE) edit set image "quay.io/dynatrace/dynatrace-operator:snapshot"=$(BRANCH_IMAGE)
	$(KUSTOMIZE) build config/deploy/openshift | oc apply -f -
