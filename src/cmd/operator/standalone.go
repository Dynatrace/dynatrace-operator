package main

import "github.com/Dynatrace/dynatrace-operator/src/standalone"

func startStandAloneInit() error {
	standaloneRunner, err := standalone.NewRunner()
	if err != nil {
		return err
	}
	return standaloneRunner.Run()
}
