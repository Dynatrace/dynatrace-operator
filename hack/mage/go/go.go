package golang

import (
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Format runs go fmt
func Format() {
	//sh.Run("go", "fmt", "./...")
	sh.Exec(nil, os.Stdout, os.Stdout, "go", "fmt", "./...")
}

// Vet runs go fmt
func Vet() {
	//sh.Run("go", "vet", "./...")
	sh.Exec(nil, os.Stdout, os.Stdout, "go", "vet", "./...")
}

// Lint lints the go code
func Lint() {
	mg.Deps(Format, Vet)
	sh.Exec(nil, os.Stdout, os.Stdout, "gci", "-w", ".")
	sh.Exec(nil, os.Stdout, os.Stdout, "golangci-lint", "run", "--build-tags", "integration,containers_image_storage_stub", "--timeout", "300s")
}

// Test runs go unit tests and writes the coverprofile to cover.out
func Test() {
	sh.Exec(nil, os.Stdout, os.Stdout, "make", "go/test")
}

// Run the Operator using the configured Kubernetes cluster in ~/.kube/config.
func Run() {
	//go/run: manifests/kubernetes manifests/openshift go/fmt go/vet
	mg.Deps(Format, Vet)
	env := map[string]string{
		"RUN_LOCAL":     "true",
		"POD_NAMESPACE": "dynatrace",
	}
	sh.Exec(env, os.Stdout, os.Stdout, "go", "run", "./src/cmd/operator/")
}

//BuildManager builds the Operators binary and writes it to bin/manager
func BuildManager() {
	//go/build/manager: manifests/crd/generate go/fmt go/vet
	mg.Deps(Format, Vet)
	sh.Exec(nil, os.Stdout, os.Stdout, "go", "build", "-o", "bin/manager", "./src/cmd/operator/")
}

//BuildManagerAmd64 builds the Operators binary specifically for AMD64 and writes it to bin/manager
func BuildManagerAmd64() {
	//go/build/manager/amd64: manifests/crd7generate go/fmt go/vet
	mg.Deps(Format, Vet)
	env := map[string]string{
		"GOOS":   "linux",
		"GOARCH": "amd64",
	}
	sh.Exec(env, os.Stdout, os.Stdout, "go", "build", "-o", "bin/manager-amd64", "./src/cmd/operator/")
}
