package main

import (
	"github.com/Dynatrace/dynatrace-operator/src/standalone"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"
)

func startStandAloneInit() error {
	unix.Umask(0000)
	standaloneRunner, err := standalone.NewRunner(afero.NewOsFs())
	if err != nil {
		return err
	}
	return standaloneRunner.Run()
}
