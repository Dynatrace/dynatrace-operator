## Generates SBOM of binary
release/gen-sbom: prerequisites/cyclonedx-gomod
	cyclonedx-gomod app -licenses -assert-licenses -json -main cmd/ -output dynatrace-operator-bin-sbom.cdx.json
