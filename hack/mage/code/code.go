package code

import (
	"github.com/Dynatrace/dynatrace-operator/hack/mage/crd"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Format runs go fmt
func Format() error {
	//sh.Exec(nil, os.Stdout, os.Stdout, "go", "fmt", "./...")
	return sh.Run("go", "fmt", "./...")
}

// Vet runs go fmt
func Vet() error {
	return sh.Run("go", "vet", "./...")
}

// Lint lints the go code
func Lint() error {
	mg.SerialDeps(Format, Vet)
	if err := sh.Run("gci", "-w", "."); err != nil {
		return err
	}
	return sh.Run("golangci-lint", "run", "--build-tags", "integration,containers_image_storage_stub", "--timeout", "300s")
}

// Test runs go unit tests and writes the coverprofile to cover.out
func Test() error {
	//go/test: manifests/kubernetes manifests/openshift go/fmt
	mg.SerialDeps(Format)
	return sh.Run("make", "go/test")
}

// Run the Operator using the configured Kubernetes cluster in ~/.kube/config.
func Run() error {
	//go/run: manifests/kubernetes manifests/openshift go/fmt go/vet
	mg.SerialDeps(Format, Vet)
	env := map[string]string{
		"RUN_LOCAL":     "true",
		"POD_NAMESPACE": "dynatrace",
	}
	return sh.RunWith(env, "go", "run", "./src/cmd/operator/")
}

//BuildManager builds the Operators binary and writes it to bin/manager
func BuildManager() error {
	mg.SerialDeps(crd.CrdGenerate, Format, Vet)
	return sh.Run("go", "build", "-o", "bin/manager", "./src/cmd/operator/")
}

//BuildManagerAmd64 builds the Operators binary specifically for AMD64 and writes it to bin/manager
func BuildManagerAmd64() error {
	mg.SerialDeps(crd.CrdGenerate, Format, Vet)
	env := map[string]string{
		"GOOS":   "linux",
		"GOARCH": "amd64",
	}
	return sh.RunWith(env, "go", "build", "-o", "bin/manager-amd64", "./src/cmd/operator/")
}
