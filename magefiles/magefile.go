//go:build mage

// Main mage package.
package main

import (
	"fmt"

	//mage:import bundle
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/bundle"
	//mage:import manifests
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/crd"
	//mage:import code
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/go"
	"github.com/Dynatrace/dynatrace-operator/hack/mage/utils"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

func Deps() error {
	return nil
}

// Build the operator image and pushes it to quay with a snapshot tag
func Build() error {
	commandPath, err := utils.GetCommand("kustomize")
	if err != nil {
		return err
	}
	fmt.Println(commandPath)

	commandPath, err = utils.GetCommand("controller-gen")
	if err != nil {
		return err
	}
	fmt.Println(commandPath)
	return nil
}

// Install dependencies
func Install(name string) error {
	fmt.Println(name)
	return nil
}

// Clean up after yourself
func Clean() {
	fmt.Println("Cleaning...")

	files := []string{
		"checkout",
		"config/manifests/bases/dynatrace-operator.clusterserviceversion.yaml",
		"config/helm/chart/default/generated/dynatrace-operator-crd.yaml",
		"config/helm/chart/default/Chart.yaml",
		"config/deploy/openshift/openshift-all.yaml",
		"config/deploy/openshift/kustomization.yaml",
		"config/deploy/kubernetes/kubernetes-all.yaml",
		"config/crd/bases/dynatrace.com_dynakubes.yaml",
	}
	sh.Run("git", files...)
}
