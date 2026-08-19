// Copyright Dynatrace LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	stdLog "log"
	"os"

	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	"github.com/Dynatrace/dynatrace-operator/cmd/certgen"
	"github.com/Dynatrace/dynatrace-operator/cmd/crdstoragemigration"
	csiInit "github.com/Dynatrace/dynatrace-operator/cmd/csi/init"
	"github.com/Dynatrace/dynatrace-operator/cmd/csi/livenessprobe"
	csiProvisioner "github.com/Dynatrace/dynatrace-operator/cmd/csi/provisioner"
	"github.com/Dynatrace/dynatrace-operator/cmd/csi/registrar"
	csiServer "github.com/Dynatrace/dynatrace-operator/cmd/csi/server"
	"github.com/Dynatrace/dynatrace-operator/cmd/metadata"
	"github.com/Dynatrace/dynatrace-operator/cmd/operator"
	startupProbe "github.com/Dynatrace/dynatrace-operator/cmd/startupprobe"
	supportArchive "github.com/Dynatrace/dynatrace-operator/cmd/supportarchive"
	"github.com/Dynatrace/dynatrace-operator/cmd/troubleshoot"
	"github.com/Dynatrace/dynatrace-operator/cmd/webhook"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = logd.Get().WithName("main")

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dynatrace-operator",
		RunE: rootCommand,
	}

	return cmd
}

func rootCommand(_ *cobra.Command, _ []string) error {
	return errors.New("operator binary must be called with one of the subcommands")
}

func main() {
	ctrl.SetLogger(logd.Get().Logger)
	stdLog.SetOutput(&log)

	cmd := newRootCommand()

	cmd.AddCommand(
		webhook.New(),
		operator.New(),
		crdstoragemigration.New(),
		certgen.New(),
		troubleshoot.New(),
		supportArchive.New(),
		startupProbe.New(),
		csiInit.New(),
		csiProvisioner.New(),
		csiServer.New(),
		livenessprobe.New(),
		registrar.New(),
		bootstrapper.New(),
		metadata.New(),
	)

	err := cmd.Execute()
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
}
