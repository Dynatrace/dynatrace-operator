package bundle

import (
	"os"

	"github.com/magefile/mage/sh"
)

// Bundle generates bundle manifests and metadata, then validates generated files
func Bundle() {
	sh.Exec(nil, os.Stdout, os.Stdout, "make", "bundle")
}

// BundleMinimal generates bundle manifests and metadata, validates generated files and removes everything but the CSV file
func BundleMinimal() {
	sh.Exec(nil, os.Stdout, os.Stdout, "make", "bundle/minimal")
}

// BundleBuild builds the docker image used for OLM deployment
func BundleBuild() {
	sh.Exec(nil, os.Stdout, os.Stdout, "make", "bundle/build")
}
