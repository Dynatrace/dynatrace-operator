-include config.mk
-include ../prerequisites.mk

## Generates a CRD in config/crd/bases
manifests/crd/generate: prerequisites/controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

## Generates a CRD in config/crd and then applies it to a cluster using kubectl
manifests/crd/install: manifests/crd/generate prerequisites/kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

## Generates a CRD in config/crd to remove it from a cluster using kubectl
manifests/crd/uninstall: manifests/crd/generate prerequisites/kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

## Builds a CRD and puts it with the Helm charts
manifests/crd/helm: manifests/crd/generate
	# Build crd
	mkdir -p "$(HELM_CRD_DIR)"
	$(KUSTOMIZE) build config/crd > $(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)

	# Copy crd to CHART PATH
	mkdir -p "$(HELM_GENERATED_DIR)"
	cp "$(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)" "$(HELM_GENERATED_DIR)"

## Generates a Kubernetes specific manifest including a CRD
manifests/kubernetes/crd: manifests/crd/generate prerequisites/controller-gen prerequisites/kustomize
	# Create directories for manifests if they do not exist
	mkdir -p config/deploy/kubernetes

	# Generate kubernetes.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set platform="kubernetes" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set operator.image="$(MASTER_IMAGE)" > "$(KUBERNETES_OTHERS_YAML)"

	grep -v 'app.kubernetes.io/managed-by' "$(KUBERNETES_OTHERS_YAML)"  > config/deploy/kubernetes/tmp.yaml
	grep -v 'helm.sh' config/deploy/kubernetes/tmp.yaml > "$(KUBERNETES_OTHERS_YAML)"
	rm config/deploy/kubernetes/tmp.yaml

	$(KUSTOMIZE) build config/crd | cat - "$(KUBERNETES_OTHERS_YAML)" > "$(KUBERNETES_CRD_AND_OTHERS_YAML)"

## Generates a OpenShift specific manifest including a CRD
manifests/openshift/crd: manifests/crd/generate prerequisites/controller-gen prerequisites/kustomize
	# Create directories for manifests if they do not exist
	mkdir -p config/deploy/openshift

	# Generate openshift.yaml
	helm template dynatrace-operator config/helm/chart/default \
		--namespace dynatrace \
		--set platform="openshift" \
		--set manifests=true \
		--set olm="${OLM}" \
		--set autoCreateSecret=false \
		--set createSecurityContextConstraints="true" \
		--set operator.image="$(MASTER_IMAGE)" > "$(OPENSHIFT_OTHERS_YAML)"

	grep -v 'app.kubernetes.io/managed-by' "$(OPENSHIFT_OTHERS_YAML)"  > config/deploy/kubernetes/tmp.yaml
	grep -v 'helm.sh' config/deploy/kubernetes/tmp.yaml > "$(OPENSHIFT_OTHERS_YAML)"
	rm config/deploy/kubernetes/tmp.yaml

	$(KUSTOMIZE) build config/crd | cat - "$(OPENSHIFT_OTHERS_YAML)" > "$(OPENSHIFT_CRD_AND_OTHERS_YAML)"

## Generates manifests for Kubernetes and OpenShift both including a CRD
manifests/crd: manifests/kubernetes/crd manifests/openshift/crd