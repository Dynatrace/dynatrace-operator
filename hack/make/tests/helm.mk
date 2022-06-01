## Unit tests the Helm charts
test/helm/unit:
	./hack/helm/test.sh

## Lints the Helm charts
test/helm/lint:
	./hack/helm/lint.sh

## Lints and then unit tests the Helm charts
test/helm: test/helm/lint test/helm/unit
