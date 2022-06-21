package config

import (
	"fmt"
	"os"
	"regexp"

	"github.com/magefile/mage/sh"
)

var (
	CRD_OPTIONS = "crd:crdVersions=v1"
	OLM         = false

	DYNATRACE_OPERATOR_CRD_YAML = "dynatrace-operator-crd.yaml"

	HELM_CHART_DEFAULT_DIR = "config/helm/chart/default/"
	HELM_GENERATED_DIR     = HELM_CHART_DEFAULT_DIR + "generated/"
	HELM_TEMPLATES_DIR     = HELM_CHART_DEFAULT_DIR + "templates/"
	HELM_CRD_DIR           = HELM_TEMPLATES_DIR + "Common/crd/"

	MANIFESTS_DIR = "config/deploy/"

	KUBERNETES_CORE_YAML      = MANIFESTS_DIR + "kubernetes/kubernetes.yaml"
	KUBERNETES_CSIDRIVER_YAML = MANIFESTS_DIR + "kubernetes/kubernetes-csidriver.yaml"
	KUBERNETES_OLM_YAML       = MANIFESTS_DIR + "kubernetes/kubernetes-olm.yaml"
	KUBERNETES_ALL_YAML       = MANIFESTS_DIR + "kubernetes/kubernetes-all.yaml"

	OPENSHIFT_CORE_YAML      = MANIFESTS_DIR + "openshift/openshift.yaml"
	OPENSHIFT_CSIDRIVER_YAML = MANIFESTS_DIR + "openshift/openshift-csidriver.yaml"
	OPENSHIFT_OLM_YAML       = MANIFESTS_DIR + "openshift/openshift-olm.yaml"
	OPENSHIFT_ALL_YAML       = MANIFESTS_DIR + "openshift/openshift-all.yaml"

	CHART_VERSION = ""

	MASTER_IMAGE = "quay.io/dynatrace/dynatrace-operator:snapshot"
)

func envInit() {
	crd, found := os.LookupEnv("CRD_OPTIONS")
	if found {
		CRD_OPTIONS = crd
	}

	olm, found := os.LookupEnv("OLM")
	if found {
		if olm == "true" {
			OLM = true
		}
	}
}

func branchInit() error {
	//printChartVersion := func() { fmt.Printf("chart: %s\n", CHART_VERSION) }
	//defer printChartVersion()

	branch, err := sh.Output("git", "branch", "--show-current")
	if err != nil {
		return err
	}
	//fmt.Printf("branch: %s\n", branch)

	release, err := regexp.Compile("^release-(.*)")
	if err != nil {
		return err
	}

	//matched, err := regexp.MatchString("^release-", branch)
	matched := release.MatchString(branch)
	if matched {
		chartVersion, err := sh.Output("/bin/bash", "-c", "grep \"^version:\" "+HELM_CHART_DEFAULT_DIR+"Chart.yaml | grep snapshot")
		if err != nil {
			return err
		}
		//fmt.Printf("version: %s\n", chartVersion)
		if chartVersion == "" {
			return nil
		}

		branchVersion := release.FindStringSubmatch(branch)
		CHART_VERSION = branchVersion[1] + ".0"
		return nil
	}

	if branch == "master" {
		// if the current branch is the master branch
		CHART_VERSION = "0.0.0-snapshot"
		return nil
	}

	return nil
}

// Init initialize helm version and other variables
func Init() {
	envInit()
	branchInit()
}

// Show the current settings
func Show() {
	fmt.Printf("%-16s: %v\n", "CRD_OPTIONS", CRD_OPTIONS)
	fmt.Printf("%-16s: %v\n", "OLM", OLM)
	fmt.Printf("%-16s: %v\n", "CHART_VERSION", CHART_VERSION)
}
