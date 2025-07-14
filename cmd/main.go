/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"os"

	"github.com/Dynatrace/dynatrace-operator/cmd/bootstrapper"
	csiInit "github.com/Dynatrace/dynatrace-operator/cmd/csi/init"
	"github.com/Dynatrace/dynatrace-operator/cmd/csi/livenessprobe"
	csiProvisioner "github.com/Dynatrace/dynatrace-operator/cmd/csi/provisioner"
	"github.com/Dynatrace/dynatrace-operator/cmd/csi/registrar"
	csiServer "github.com/Dynatrace/dynatrace-operator/cmd/csi/server"
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

var (
	log = logd.Get().WithName("main")
)

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
	ctrl.SetLogger(log.Logger)

	cmd := newRootCommand()

	cmd.AddCommand(
		webhook.New(),
		operator.New(),
		troubleshoot.New(),
		supportArchive.New(),
		startupProbe.New(),
		csiInit.New(),
		csiProvisioner.New(),
		csiServer.New(),
		livenessprobe.New(),
		registrar.New(),
		bootstrapper.New(),
	)

	err := cmd.Execute()
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
}
