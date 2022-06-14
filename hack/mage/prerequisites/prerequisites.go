package prerequisites

import (
	"github.com/Dynatrace/dynatrace-operator/hack/mage/utils"
	"github.com/magefile/mage/sh"
)

// Kustomize installs kustomize command
func Kustomize() error {
	_, err := utils.GetCommand("kustomize")
	if err != nil {
		return sh.Run("hack/build/command.sh", "kustomize", "sigs.k8s.io/kustomize/kustomize/v3@v3.5.4")
	}
	return nil
}

// ControllerGen installs controller-gen command
func ControllerGen() error {
	_, err := utils.GetCommand("controller-gen")
	if err != nil {
		return sh.Run("hack/build/command.sh", "controller-gen", "sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.0")
	}
	return nil
}

func SetupPreCommit() error {
	err := sh.Run("go", "install", "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.46.2")
	if err != nil {
		return err
	}
	err = sh.Run("go", "install", "github.com/daixiang0/gci@v0.3.3")
	if err != nil {
		return err
	}
	err = sh.Run("go", "install", "golang.org/x/tools/cmd/goimports@v0.1.10")
	if err != nil {
		return err
	}
	err = sh.Run("cp", "./.github/pre-commit", "./.git/hooks/pre-commit")
	if err != nil {
		return err
	}
	return sh.Run("chmod", "+x", "./.git/hooks/pre-commit")
}
