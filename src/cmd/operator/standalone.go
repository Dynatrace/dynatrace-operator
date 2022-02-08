package main

import (
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	"github.com/spf13/afero"
)

func startStandAloneInit() error {
	standaloneRunner, err := standalone.NewRunner(afero.NewOsFs())
	if err != nil {
		return err
	}
	return standaloneRunner.Run()
}
