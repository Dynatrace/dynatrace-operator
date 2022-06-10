## Sets the Helm Charts version and appVersion
helm/version:
ifneq ($(CHART_VERSION),)
	sed "s/^version: .*/version: $(CHART_VERSION)/" $(HELM_CHART_DEFAULT_DIR)/Chart.yaml >  $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output
	mv $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output $(HELM_CHART_DEFAULT_DIR)/Chart.yaml
	sed "s/^appVersion: .*/appVersion: $(CHART_VERSION)/" $(HELM_CHART_DEFAULT_DIR)/Chart.yaml >  $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output
	mv $(HELM_CHART_DEFAULT_DIR)/Chart.yaml.output $(HELM_CHART_DEFAULT_DIR)/Chart.yaml
endif
