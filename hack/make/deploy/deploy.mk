ENABLE_CSI ?= true
DEBUG_LOGS ?= true
PLATFORM ?= "kubernetes"
HELM_CHART ?= config/helm/chart/default

## Deploy the operator without the csi-driver
deploy/no-csi:
	@make ENABLE_CSI=false $(@D)

deploy/fips:
	@make IMAGE_URI="$(IMAGE_URI)"-fips $(@D)

## Deploy the operator with csi-driver
deploy: manifests/crd/helm
	helm upgrade dynatrace-operator $(HELM_CHART) \
			--install \
			--namespace dynatrace \
			--create-namespace \
			--atomic \
			--set installCRD=true \
			--set csidriver.enabled=$(ENABLE_CSI) \
			--set manifests=true \
			--set image=$(IMAGE_URI) \
			--set debugLogs=$(DEBUG_LOGS) \
			--set debug=$(DEBUG)

## Undeploy the current operator installation
undeploy: cleanup/custom-resources
	helm uninstall dynatrace-operator \
			--namespace dynatrace

## Remove all Dynatrace Operator resources from the cluster
cleanup: cleanup/custom-resources
	kubectl delete namespace dynatrace --ignore-not-found
	@echo "Removing Dynatrace Operator cluster-scoped resources"
	@kubectl api-resources --verbs=list -o name --namespaced=false | \
		xargs -I {} sh -c \
		"kubectl delete {} --ignore-not-found -l app.kubernetes.io/name=dynatrace-operator 2>&1 | \
		grep -v 'No resources found'" || true

	@echo -n "Removing Dynatrace Operator secrets and configmaps from all namespaces "
	@for ns in $$(kubectl get ns -o jsonpath="{.items[*].metadata.name}"); do \
		kubectl delete secret,cm -l app.kubernetes.io/name=dynatrace-operator --ignore-not-found -n $$ns > /dev/null 2>&1 || true; \
		printf '.'; \
	done
	@echo " done"

cleanup/custom-resources:
	kubectl delete dynakube --all -n dynatrace || true
	kubectl delete edgeconnect --all -n dynatrace || true
	kubectl -n dynatrace wait pod --for=delete -l app.kubernetes.io/managed-by=dynatrace-operator --timeout=300s
