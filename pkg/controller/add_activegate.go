package controller

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/controller/activegate"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, activegate.Add)
}
