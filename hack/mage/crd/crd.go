package crd

import (
	"github.com/Dynatrace/dynatrace-operator/hack/mage/utils"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os"
)

// CrdGenerate generates a CRD in config/crd/bases
func CrdGenerate() {
	//manifests/crd/generate: prerequisites/controller-gen
	crdOptions := os.Getenv("CRD_OPTIONS")
	if crdOptions != "" {
		crdOptions = "crd:crdVersions=v1"
	}

	controllerGen, err := utils.GetCommand("controller-gen")
	if err != nil {
		return
	}

	sh.Exec(nil, os.Stdout, os.Stdout, controllerGen, crdOptions, "paths=\"./...\"", "output:crd:artifacts:config=config/crd/bases")
}

// CrdInstall generates a CRD in config/crd and then applies it to a cluster using kubectl
func CrdInstall() {
	mg.Deps(CrdGenerate)

	kustomize, err := utils.GetCommand("kustomize")
	if err != nil {
		return
	}

	sh.Exec(nil, os.Stdout, os.Stdout, "/bin/bash", "-c", kustomize+" build config/crd | kubectl apply -f -")
}

// CrdUninstall generates a CRD in config/crd to remove it from a cluster using kubectl
func CrdUninstall() {
	mg.Deps(CrdGenerate)

	kustomize, err := utils.GetCommand("kustomize")
	if err != nil {
		return
	}

	sh.Exec(nil, os.Stdout, os.Stdout, "/bin/bash", "-c", kustomize+" build config/crd | kubectl delete -f -")
}

// CrdHelm builds a CRD and puts it with the Helm charts
func CrdHelm() {
	/*mg.Deps(CrdGenerate)

	kustomize, err := utils.GetCommand("kustomize")
	if err != nil {
		return
	}*/

	sh.Exec(nil, os.Stdout, os.Stdout, "make", "manifests/crd/helm")
}
