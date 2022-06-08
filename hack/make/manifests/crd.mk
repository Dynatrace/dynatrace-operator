-include config.mk
-include ../prerequisites.mk

helm/version:
ifneq ($(CHART_VERSION),)
	sed "s/^version: .*/version: $(CHART_VERSION)/" $(HELM_CHART_DEFAULT_DIR)/Chart.yaml >  $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output
	mv $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output $(HELM_CHART_DEFAULT_DIR)/Chart.yaml
	sed "s/^appVersion: .*/appVersion: $(CHART_VERSION)/" $(HELM_CHART_DEFAULT_DIR)/Chart.yaml >  $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output
	mv $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output $(HELM_CHART_DEFAULT_DIR)/Chart.yaml
endif

## Generates a CRD in config/crd/bases
manifests/crd/generate: prerequisites/controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=config/crd/bases

## Generates a CRD in config/crd and then applies it to a cluster using kubectl
manifests/crd/install: prerequisites/kustomize manifests/crd/generate
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

## Generates a CRD in config/crd to remove it from a cluster using kubectl
manifests/crd/uninstall: prerequisites/kustomize manifests/crd/generate
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

## Builds a CRD and puts it with the Helm charts
manifests/crd/helm: helm/version prerequisites/kustomize manifests/crd/generate
	# Build crd
	mkdir -p "$(HELM_CRD_DIR)"
	$(KUSTOMIZE) build config/crd > $(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)

	# Copy crd to CHART PATH
	mkdir -p "$(HELM_GENERATED_DIR)"
	cp "$(MANIFESTS_DIR)/kubernetes/$(DYNATRACE_OPERATOR_CRD_YAML)" "$(HELM_GENERATED_DIR)"
