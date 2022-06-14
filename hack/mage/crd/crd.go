package crd

import (
	"fmt"
	"os"

	"github.com/Dynatrace/dynatrace-operator/hack/mage/config"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/helm"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/prerequisites"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/utils"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// golang command pipes
// https://stackoverflow.com/questions/10781516/how-to-pipe-several-commands-in-go

// python
// https://docs.python.org/3/library/subprocess.html#subprocess.run

// CrdGenerate generates a CRD in config/crd/bases
func CrdGenerate() error {
	mg.SerialDeps(prerequisites.ControllerGen)

	crdOptions := os.Getenv("CRD_OPTIONS")
	if crdOptions == "" {
		crdOptions = "crd:crdVersions=v1"
	}

	controllerGen, err := utils.GetCommand("controller-gen")
	if err != nil {
		return err
	}

	// crdOptions has to be split by ' ' to be properly handled by sh.Run()
	return sh.Run(controllerGen, crdOptions, "paths=\"./...\"", "output:crd:artifacts:config=config/crd/bases")
}

// CrdInstall generates a CRD in config/crd and then applies it to a cluster using kubectl
func CrdInstall() error {
	mg.SerialDeps(prerequisites.Kustomize, CrdGenerate)

	kustomize, err := utils.GetCommand("kustomize")
	if err != nil {
		return err
	}

	return sh.Run("/bin/bash", "-c", kustomize+" build config/crd | kubectl apply -f -")
}

// CrdUninstall generates a CRD in config/crd to remove it from a cluster using kubectl
func CrdUninstall() error {
	mg.SerialDeps(prerequisites.Kustomize, CrdGenerate)

	kustomize, err := utils.GetCommand("kustomize")
	if err != nil {
		return err
	}

	return sh.Run("/bin/bash", "-c", kustomize+" build config/crd | kubectl delete -f -")
}

// CrdHelm builds a CRD and puts it with the Helm charts
func CrdHelm() error {
	mg.SerialDeps(helm.Version, prerequisites.Kustomize, CrdGenerate)

	kustomize, err := utils.GetCommand("kustomize")
	if err != nil {
		return err
	}

	if err := sh.Run("mkdir", "-p", config.HELM_CRD_DIR); err != nil {
		return err
	}

	crdYamlFile, err := os.OpenFile(config.MANIFESTS_DIR+"kubernetes/"+config.DYNATRACE_OPERATOR_CRD_YAML, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return nil
	}
	ran, err := sh.Exec(nil, crdYamlFile, os.Stderr, kustomize, "build", "config/crd")
	crdYamlFile.Close()
	if !ran {
		return fmt.Errorf(kustomize + " not executed")
	}
	if err != nil {
		return nil
	}

	if err := sh.Run("mkdir", "-p", config.HELM_GENERATED_DIR); err != nil {
		return err
	}
	if err := sh.Run("cp", config.MANIFESTS_DIR+"kubernetes/"+config.DYNATRACE_OPERATOR_CRD_YAML, config.HELM_GENERATED_DIR); err != nil {
		return err
	}
	return nil
}
