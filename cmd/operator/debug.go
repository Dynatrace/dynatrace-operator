package main

import (
	"context"
	"fmt"
	"os"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type startupInfo struct {
	cfg           *rest.Config
	namespace     string
	signalHandler context.Context
}

type subCommand func(string, *rest.Config) (manager.Manager, error)

func startWebhookAndBootstrapperIfDebugFlagSet(info startupInfo) {
	if isDebugFlagSet() {
		log.Info("debug mode enabled")
		log.Info("starting webhook and bootstrapper")
		go startBootstrapperManager(info)
		go startWebhookManager(info)
	}
}

func isDebugFlagSet() bool {
	debugFlag := os.Getenv("DEBUG_OPERATOR")
	return debugFlag == "true"
}

func startBootstrapperManager(info startupInfo) {
	startComponent("webhook-bootstrapper", info)
}

func startWebhookManager(info startupInfo) {
	startComponent("webhook-server", info)
}

func startComponent(name string, info startupInfo) {
	subCmd, err := getSubcommand(name)
	if err != nil {
		return
	}
	startSubCommand(name, subCmd, &info)
}

func getSubcommand(name string) (subCommand, error) {
	subcmdFn, hasSubCommand := subcmdCallbacks[name]
	if !hasSubCommand {
		log.Error(errBadSubcmd, "unknown command", "command", "webhook-server")
		return subcmdFn, errBadSubcmd
	}
	return subcmdFn, nil
}

func startSubCommand(name string, cmd subCommand, startInfo *startupInfo) {
	mgr, err := cmd(startInfo.namespace, startInfo.cfg)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info(fmt.Sprintf("starting manager '%s'", name))
	if err := mgr.Start(startInfo.signalHandler); err != nil {
		log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
