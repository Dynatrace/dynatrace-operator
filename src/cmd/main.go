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
	"github.com/Dynatrace/dynatrace-operator/src/cmd/csi"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/operator"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/standalone"
	"github.com/Dynatrace/dynatrace-operator/src/cmd/webhook"
	"github.com/Dynatrace/dynatrace-operator/src/kubesystem"
	"github.com/Dynatrace/dynatrace-operator/src/logger"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	log = logger.NewDTLogger().WithName("main")
)

const (
	envPodNamespace = "POD_NAMESPACE"
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "dynatrace-operator",
		RunE: rootCommand,
	}

	return cmd
}

func addWebhookCommand(cmd *cobra.Command) {
	webhookCommandBuilder := webhook.NewWebhookCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetIsDeployedViaOlm(kubesystem.DeployedViaOLM()).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())

	cmd.AddCommand(webhookCommandBuilder.Build())
}

func addOperatorCommand(cmd *cobra.Command) {
	operatorCommandBuilder := operator.NewOperatorCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetIsDeployedViaOlm(kubesystem.DeployedViaOLM()).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())

	cmd.AddCommand(operatorCommandBuilder.Build())
}

func addCsiCommand(cmd *cobra.Command) {
	csiCommandBuilder := csi.NewCsiCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())

	cmd.AddCommand(csiCommandBuilder.Build())
}

func addStandaloneCommand(cmd *cobra.Command) {
	cmd.AddCommand(standalone.NewStandaloneCommand())
}

func rootCommand(_ *cobra.Command, _ []string) error {
	return errors.New("operator binary must be called with one of the subcommands")
}

func main() {
	ctrl.SetLogger(log)
	cmd := newRootCommand()

	addWebhookCommand(cmd)
	addOperatorCommand(cmd)
	addCsiCommand(cmd)
	addStandaloneCommand(cmd)

	err := cmd.Execute()
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
}
