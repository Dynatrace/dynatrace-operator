OAPI_CONFIG_DIR := api/oapi
OAPI_SYNC_CONFIG := $(OAPI_CONFIG_DIR)/sync-config.yaml
OAPI_GENERATOR_CONFIG := $(OAPI_CONFIG_DIR)/generator-config.yaml
OAPI_IGNORE_FILE := $(OAPI_CONFIG_DIR)/.openapi-generator-ignore
OAPI_MODELS_FOR_TAG := hack/make/bin/oapi-models-for-tag.sh

# Defaults from generator-config.yaml (lazily evaluated, only when target runs)
OAPI_GENERATOR ?= $(shell yq -r '.generator' $(OAPI_GENERATOR_CONFIG))
OAPI_GENERATOR_VERSION ?= $(shell yq -r '.generatorVersion' $(OAPI_GENERATOR_CONFIG))
OAPI_OUTPUT_DIR ?= $(shell yq -r '.outputDir' $(OAPI_GENERATOR_CONFIG))
OAPI_ADDITIONAL_PROPS ?= $(shell yq -r '.additionalProperties // ""' $(OAPI_GENERATOR_CONFIG))
OAPI_GLOBAL_PROPS ?= $(shell yq -r '.globalProperties // ""' $(OAPI_GENERATOR_CONFIG))

# Builds the --global-property value from per-schema config.
# apis drives model auto-resolution via oapi-models-for-tag.sh — see below.
# Global is prepended; per-schema appended — last-value-wins on duplicates.
jq_global_props = .generate.globalProperties as $$gp \
  | (["$(OAPI_GLOBAL_PROPS)"] | map(select(length > 0))) + [ \
      (if $$gp.apis       and ($$gp.apis       | length) > 0 then "apis="   + ($$gp.apis   | join(":")) else empty end), \
      (if $$gp.additional and ($$gp.additional | length) > 0 then $$gp.additional                       else empty end) \
    ] | join(",")


## Generate Go SDKs from OpenAPI specs
oapi/generate: prerequisites/openapi-generator-cli
	@yq -o=json '.schemas | map(select(.generate))' "$(OAPI_SYNC_CONFIG)" \
	| jq -c '.[]' \
	| while read -r row; do \
		name=$$(echo "$$row" | jq -r '.name'); \
		pkg=$$(echo "$$row" | jq -r '.generate.packageName // .name'); \
		ver=$$(echo "$$row" | jq -r '.generate.generatorVersion // "$(OAPI_GENERATOR_VERSION)"'); \
		gen=$$(echo "$$row" | jq -r '.generate.generator // "$(OAPI_GENERATOR)"'); \
		props=$$(echo "$$row" | jq -r '.generate.additionalProperties // "$(OAPI_ADDITIONAL_PROPS)"'); \
		out=$$(echo "$$row" | jq -r '.generate.outputDir // "$(OAPI_OUTPUT_DIR)/'"$$pkg"'"'); \
		spec_url=$$(echo "$$row" | jq -r '.specUrlEnvVar // ""' | xargs -I{} printenv {} 2>/dev/null || echo ""); \
		auth_var=$$(echo "$$row" | jq -r '.authEnvVar // ""'); \
		auth_val=$$([ -n "$$auth_var" ] && printenv "$$auth_var" || echo ""); \
		auth_header=$$([ -n "$$auth_val" ] && printf 'Authorization:Bearer%%20%s' "$$auth_val" || echo ""); \
		[ -n "$$spec_url" ] || { echo "WARNING: no specUrlEnvVar set for $$name, skipping."; continue; }; \
		echo "Downloading spec for $$name from $$spec_url ..."; \
		tmp_spec=$$(mktemp); \
		if [ -n "$$auth_val" ]; then \
			curl -sSfL --location-trusted -H "Authorization: Bearer $$auth_val" "$$spec_url" -o "$$tmp_spec"; \
		else \
			curl -sSfL "$$spec_url" -o "$$tmp_spec"; \
		fi || { echo "ERROR: curl failed for $$name (HTTP error or connection problem)"; rm -f "$$tmp_spec"; continue; }; \
		[ -s "$$tmp_spec" ] || { echo "ERROR: downloaded spec for $$name is empty"; rm -f "$$tmp_spec"; continue; }; \
		jq . "$$tmp_spec" > /dev/null 2>&1 || { echo "ERROR: downloaded spec for $$name is not valid JSON (got HTML? auth redirect?)"; rm -f "$$tmp_spec"; continue; }; \
		apis=$$(echo "$$row" | jq -r '.generate.globalProperties.apis // [] | .[]'); \
		models=""; \
		for tag in $$apis; do \
			tag_models=$$($(OAPI_MODELS_FOR_TAG) "$$tmp_spec" "$$tag" | tr '\n' ' '); \
			models="$$models $$tag_models"; \
		done; \
		models=$$(echo "$$models" | tr ' ' '\n' | sort -u | grep -v '^$$' | tr '\n' ':' | sed 's/:$$//'); \
		gprops=$$(echo "$$row" | jq -r '$(jq_global_props)'); \
		gprops=$$(printf '%s' "$${models:+models=$$models$${gprops:+,}}$$gprops"); \
		echo "Generating $$name ($$gen $$ver, package: $$pkg)..."; \
		rm -rf "$$out" && mkdir -p "$$out"; \
		cp "$(OAPI_IGNORE_FILE)" "$$out/.openapi-generator-ignore"; \
		OPENAPI_GENERATOR_VERSION="$$ver" $(OPENAPI_GENERATOR_CLI) generate \
			-i "$$tmp_spec" -g "$$gen" -o "$$out" \
			$${pkg:+--package-name "$$pkg"} \
			$${props:+--additional-properties="$$props"} \
			$${gprops:+--global-property="$$gprops"} \
			$${auth_header:+--auth "$$auth_header"} \
			--skip-validate-spec \
			--minimal-update; \
		rm -f "$$tmp_spec" "$$out/.openapi-generator-ignore"; \
		echo "Done: $$out"; \
	done
