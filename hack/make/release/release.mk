## Generates SBOM of binary
release/gen-sbom: prerequisites/cyclonedx-gomod
	cyclonedx-gomod app -licenses -assert-licenses -json -main cmd/ -output dynatrace-operator-bin-sbom.cdx.json

## Generates SBOM of binary with CGO
release/gen-sbom/fips: prerequisites/cyclonedx-gomod
	CGO_ENABLED=1 cyclonedx-gomod app -licenses -assert-licenses -json -main cmd/ -output dynatrace-operator-bin-sbom.cdx.json
