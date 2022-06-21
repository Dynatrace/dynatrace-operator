//go:build mage

// Main mage package.
package main

import (
	"fmt"

	//mage:import bundle
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/bundle"
	//mage:import code
	"github.com/Dynatrace/dynatrace-operator/hack/mage/code"
	//mage:import init
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/config"
	//mage:import crd
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/crd"
	//mage:import shallow
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/deep"
	//mage:import deps
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/dependencies"
	//mage:import helm
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/helm"
	//mage:import manifests
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/manifests"
	//mage:import prerequisites
	"github.com/Dynatrace/dynatrace-operator/hack/mage/prerequisites"
	//mage:import vars
	_ "github.com/Dynatrace/dynatrace-operator/hack/mage/vars"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

var Aliases = map[string]interface{}{
	"go:lint":  code.Lint,
	"go:purge": Clean,
}

// deps installs dependencies
func Deps() error {
	mg.SerialDeps(prerequisites.SetupPreCommit, prerequisites.Kustomize, prerequisites.ControllerGen)
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

// Parameters a target with parameters. Parameters:
// <dir> - directory,
// <level> - recursion level.
func Parameters(dir string, level int) {
	fmt.Printf("Params dir: '%v' level: '%v'\n", dir, level)
}

/*func main() {
	fmt.Println("main")
}*/
