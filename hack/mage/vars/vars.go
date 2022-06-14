package vars

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/hack/mage/config"
	"github.com/magefile/mage/mg"
)

// Aaa base target, unmodified global variable
func Aaa() {
	fmt.Printf("aaa %s\n", config.CRD_OPTIONS)
}

// Bbb depends on Aaa, modifies global variable
func Bbb() {
	mg.Deps(Aaa)
	config.CRD_OPTIONS = "<changed>"
	fmt.Printf("bbb %s\n", config.CRD_OPTIONS)
}

// Ccc depends on Bbb
func Ccc() {
	mg.Deps(Bbb)
	fmt.Printf("ccc %s\n", config.CRD_OPTIONS)
}
