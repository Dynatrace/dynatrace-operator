## Generate Go SDKs from OpenAPI specs
oapi/generate: prerequisites/openapi-generator-cli prerequisites/python
	@OPENAPI_GENERATOR_CLI=$(OPENAPI_GENERATOR_CLI) $(PYTHON) hack/make/bin/oapi-generate.py
