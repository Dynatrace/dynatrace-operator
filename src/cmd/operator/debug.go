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
	"context"
	"os"

	"k8s.io/client-go/rest"
)

type startupInfo struct {
	cfg           *rest.Config
	namespace     string
	signalHandler context.Context
}

func startWebhookIfDebugFlagSet(info startupInfo) {
	if isDebugFlagSet() {
		log.Info("debug mode enabled")
		log.Info("starting webhook")
		go startWebhookManager(info)
	}
}

func isDebugFlagSet() bool {
	debugFlag := os.Getenv("DEBUG_OPERATOR")
	return debugFlag == "true"
}

func startWebhookManager(info startupInfo) {
	startComponent("webhook-server", info)
}

func startComponent(name string, startInfo startupInfo) {
	mgr, cleanUp, err := setupWebhookServer(startInfo.namespace, startInfo.cfg)
	defer cleanUp()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("starting manager", "name", name)
	if err := mgr.Start(startInfo.signalHandler); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
