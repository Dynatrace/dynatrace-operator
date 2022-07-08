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

func createWebhookCommandBuilder(deployedViaOLM bool) webhook.CommandBuilder {
	return webhook.NewWebhookCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetIsDeployedViaOlm(deployedViaOLM).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())
}

func createOperatorCommandBuilder(deployedViaOLM bool) operator.CommandBuilder {
	return operator.NewOperatorCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetIsDeployedViaOlm(deployedViaOLM).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())
}

func createCsiCommandBuilder() csi.CommandBuilder {
	return csi.NewCsiCommandBuilder().
		SetNamespace(os.Getenv(envPodNamespace)).
		SetConfigProvider(cmdConfig.NewKubeConfigProvider())
}

func rootCommand(_ *cobra.Command, _ []string) error {
	return errors.New("operator binary must be called with one of the subcommands")
}

func main() {
	ctrl.SetLogger(log)
	cmd := newRootCommand()

	podName := os.Getenv(envPodName)
	podNamespace := os.Getenv(envPodNamespace)

	clt, err := kubesystem.CreateDefaultClient()
	if err != nil {
		log.Error(err, "unable to create kube client")
		os.Exit(1)
	}
	deployedViaOLM, err := kubesystem.IsDeployedViaOLM(clt, podName, podNamespace)
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}

	cmd.AddCommand(
		createWebhookCommandBuilder(deployedViaOLM).Build(),
		createOperatorCommandBuilder(deployedViaOLM).Build(),
		createCsiCommandBuilder().Build(),
		standalone.NewStandaloneCommand(),
	)

	err = cmd.Execute()
	if err != nil {
		log.Info(err.Error())
		os.Exit(1)
	}
}
