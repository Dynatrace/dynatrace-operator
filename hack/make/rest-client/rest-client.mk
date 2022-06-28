SWAGGER_VERSION=3.0.34
DESTINATION=src/rest-client
USER=$(shell id -u)
GROUP=$(shell id -g)

## Generate a rest-client go code for example interface
generate-rest-client:
	rm -rf $(DESTINATION)
	mkdir -p $(DESTINATION)
	docker run --rm -u $(USER):$(GROUP) -v ${PWD}/$(DESTINATION):/local swaggerapi/swagger-codegen-cli-v3:$(SWAGGER_VERSION) generate -i http://petstore.swagger.io/v2/swagger.json -l go -o /local/

