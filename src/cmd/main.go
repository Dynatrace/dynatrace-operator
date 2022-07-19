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

	cmdConfig "github.com/Dynatrace/dynatrace-operator/src/cmd/config"
	csi_provisioner "github.com/Dynatrace/dynatrace-operator/src/cmd/csi-provisioner"
	csi_server "github.com/Dynatrace/dynatrace-operator/src/cmd/csi-server"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/operator"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/standalone"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = logger.NewDTLogger().WithName("main")
)

const (
	envPodNamespace = "POD_NAMESPACE"
	envPodName      = "POD_NAME"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dynatrace-operator",
		RunE: rootCommand,
	}

	return cmd
}

func createWebhookCommandBuilder() webhook.CommandBuilder {
	return webhook.NewWebhookCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetPodName(os.Getenv(envPodName)).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())
}

func createOperatorCommandBuilder() operator.CommandBuilder {
	return operator.NewOperatorCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetPodName(os.Getenv(envPodName)).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())
}

func createCsiServerCommandBuilder() csi_server.CommandBuilder {
	return csi_server.NewCsiServerCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())
}

func createCsiProvisionerCommandBuilder() csi_provisioner.CommandBuilder {
	return csi_provisioner.NewCsiProvisionerCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())
}

func rootCommand(_ *cobra.Command, _ []string) error {
	return errors.New("operator binary must be called with one of the subcommands")
}

func main() {
	ctrl.SetLogger(log)
	cmd := newRootCommand()

	cmd.AddCommand(
		createWebhookCommandBuilder().Build(),
		createOperatorCommandBuilder().Build(),
		createCsiServerCommandBuilder().Build(),
		createCsiProvisionerCommandBuilder().Build(),
		standalone.NewStandaloneCommand(),
	)

	err := cmd.Execute()
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
}
