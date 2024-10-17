# Debugging the operator

This document outlines the distinct debugging requirements for various components, providing detailed instructions for each to ensure effective troubleshooting and development.

**Important:** Read the [One-Time Setup](#one-time-setup) section before proceeding with the debugging instructions.

## Makefile Helpers

| Command                             | Description                                                                                                                                                                                               |
|-------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `make debug/build`                  | Build image with Delve debugger included.                                                                                                                                                                 |
| `make debug/deploy`                 | Install image with necessary changes to deployments. (Changes to resources, lifenessprobes, commands)                                                                                                     |
| `make debug/operator`               | Run the operator locally. Would recommend to use your IDE here instead, to have breakpoints.                                                                                                              |
| `make debug/webhook`                | Run the webhook locally. Would recommend to use your IDE here instead, to have breakpoints.                                                                                                               |
| `make debug/csi/redeploy`           | In case of code changes, closes the tunnel, rebuilds/deploys the image and opens the tunnel again.                                                                                                        |
| `make debug/tunnel/start`           | Open a tunnel from your local machine to CSI driver pod, to access debugger running in the CSI driver container. <br/>It forwards ports 40000 and 40001 to the alphabetically first CSI driver container. |
| `make debug/tunnel/stop`            | Stop the tunnel from local machine to CSI driver pod.                                                                                                                                                     |
| `make debug/telepresence/install`   | Install and setup Telepresence to intercept requests to the webhook and forward them to your local machine.                                                                                               |
| `make debug/telepresence/uninstall` | Stop Telepresence and remove all changes made to the cluster.                                                                                                                                             |

## Debug Instructions

### Summary

| Component          | What is possible                                                                                                            |
|--------------------|-----------------------------------------------------------------------------------------------------------------------------|
| Operator Main Code | Run operator locally on your machine. Debug from within your IDE.                                                           |
| Webhook            | Run webhook locally and forward cluster traffic using Telepresence. Debug from within your IDE.                             |
| CSI-Driver Server  | Run CSI driver executables on the node with debugger included. <br/> Forward debugging port and debug from within your IDE. |

### Operator Main Code

#### Context

The operator can be run locally on your machine.
As the operator only sends requests to the Kubernetes API, but never receives any, it can be run locally without any issues.

#### Setup

1. Deploy the operator as usual:

```shell
make deploy/helm
```

2. Scale down the cluster operator:

```shell
kubectl -n dynatrace scale --replicas 0 deployment/dynatrace-operator
```

#### Run

Select the 'Debug Operator' configuration in your IDE and start the debugger.
Breakpoints will be respected.

#### Teardown

1. Scale up the cluster operator:

```shell
kubectl -n dynatrace scale --replicas 1 deployment/dynatrace-operator
```

### Webhook

#### Context

The webhook can be run locally, however, as it is a webserver, kubernetes requests have to be forwarded to your local machine.
This can be achieved using Telepresence. It does this by adding a sidecar container to the webhook pod, which forwards all requests to your local machine.
The security context of the webhook pod has to be removed, as Telepresence would copy it and fail.

#### Setup

1. Build the debug image, to include the Delve debugger for the CSI driver (requirement for step 2):

```shell
make debug/build
```

2. Deploy the debug image with necessary changes to the webhook deployment (enables the debugger for the CSI driver too):

```shell
make debug/deploy
```

3. Install Telepresence in the cluster and connect to it:

```shell
make debug/telepresence/install
```

#### Run

Select the 'Debug Webhook' configuration in your IDE and start the debugger.
Breakpoints will be respected.

#### Teardown

1. Stop Telepresence and remove all changes made to the cluster:

```shell
make debug/telepresence/uninstall
```

2. Deploy without debugging changes:

```shell
make deploy/helm
```

### CSI-Driver Server

#### Context

Due to the file handling operations of the CSI driver, it is not possible to run the CSI driver locally.
However, the CSI driver can be run on the node with the debugger included.
The debugging port has to be forwarded to your local machine, where the IDE can attach to the running process in the CSI driver container.

#### Setup

1. Build the debug image, to include the Delve debugger:

```shell
make debug/build
```

2. Deploy the debug image with necessary changes to the CSI driver deployment:

```shell
make debug/deploy
```

3. Open a tunnel from your local machine to the CSI driver pod:

```shell
make debug/tunnel/start
```

#### Run

Select the 'Debug CSI driver (server)' configuration in your IDE and start the debugger.
Breakpoints will be respected.

If changes are made to the CSI driver, the image has to be rebuilt and redeployed:

```shell
make debug/csi/redeploy
```

After redeployment, select the 'Debug CSI driver (server)' configuration in your IDE again and start the debugger.

#### Teardown

1. Stop the tunnel from your local machine to the CSI driver pod:

```shell
make debug/tunnel/stop
```

2. Deploy without debugging changes:

```shell
make deploy/helm
```

### CSI-Driver Provisioner

The same as for the CSI-Driver Server, but use the 'Debug CSI driver (provisioner)' configuration in your IDE.

#### Context

The debugging process is the same as for the CSI-Driver Server, but the debugging port changes from 40000 to 40001.
That's why the 'Debug CSI driver (provisioner)' configuration has to be used in your IDE.

### Init Container

Debugging the init container is not possible, due to port-forwarding limitations of Kubernetes.

## One-Time Setup

For the above debugging steps to work, Telepresence has to be installed and configurations have to be set up in your IDE.
This section has to be done only once.

### Telepresence

Telepresence has to be installed on your local machine to forward requests to the webhook service to your local machine.
For installation instructions, refer to the [Telepresence documentation](https://www.telepresence.io/docs/install/client).

### IntelliJ

#### 1. Install environment file plugin

The following plugin is required to deal with environment files: [EnvFile](https://plugins.jetbrains.com/plugin/7861-envfile)

#### 2. Create debug configuration

1. Create a `.run` directory and create the following files:
    - `Debug CSI driver (provisioner).run.xml`
    - `Debug CSI driver (server).run.xml`
    - `Debug Operator.run.xml`
    - `Debug Webhook.run.xml`

2. Copy the following content into the respective files:

`Debug CSI driver (provisioner).run.xml`:

```xml
<component name="ProjectRunConfigurationManager">
<configuration default="false" name="Debug CSI driver (provisioner)" type="GoRemoteDebugConfigurationType" factoryName="Go Remote" port="40001">
<option name="disconnectOption" value="LEAVE" />
<disconnect value="LEAVE" />
<method v="2" />
</configuration>
</component>
```

`Debug CSI driver (server).run.xml`:

```xml
<component name="ProjectRunConfigurationManager">
    <configuration default="false" name="Debug CSI driver (server)" type="GoRemoteDebugConfigurationType" factoryName="Go Remote" port="40000">
        <option name="disconnectOption" value="LEAVE" />
        <disconnect value="LEAVE" />
        <method v="2" />
    </configuration>
</component>
```

`Debug Operator.run.xml`:

```xml
<component name="ProjectRunConfigurationManager">
    <configuration default="false" name="Debug Operator" type="GoApplicationRunConfiguration" factoryName="Go Application">
        <module name="dynatrace-operator" />
        <working_directory value="$PROJECT_DIR$" />
        <parameters value="operator" />
        <envs>
            <env name="POD_NAMESPACE" value="dynatrace" />
            <env name="RUN_LOCAL" value="true" />
        </envs>
        <EXTENSION ID="net.ashald.envfile">
            <option name="IS_ENABLED" value="false" />
            <option name="IS_SUBST" value="false" />
            <option name="IS_PATH_MACRO_SUPPORTED" value="false" />
            <option name="IS_IGNORE_MISSING_FILES" value="false" />
            <option name="IS_ENABLE_EXPERIMENTAL_INTEGRATIONS" value="false" />
            <ENTRIES>
                <ENTRY IS_ENABLED="true" PARSER="runconfig" IS_EXECUTABLE="false" />
            </ENTRIES>
        </EXTENSION>
        <kind value="DIRECTORY" />
        <package value="github.com/Dynatrace/dynatrace-operator" />
        <directory value="$PROJECT_DIR$/cmd" />
        <filePath value="$PROJECT_DIR$" />
        <method v="2" />
    </configuration>
</component>
```

`Debug Webhook.run.xml`:

```xml
<component name="ProjectRunConfigurationManager">
    <configuration default="false" name="Debug Webhook" type="GoApplicationRunConfiguration" factoryName="Go Application">
        <module name="dynatrace-operator" />
        <working_directory value="$PROJECT_DIR$" />
        <parameters value="webhook-server --certs-dir=./local/certs/" />
        <EXTENSION ID="net.ashald.envfile">
            <option name="IS_ENABLED" value="true" />
            <option name="IS_SUBST" value="false" />
            <option name="IS_PATH_MACRO_SUPPORTED" value="false" />
            <option name="IS_IGNORE_MISSING_FILES" value="false" />
            <option name="IS_ENABLE_EXPERIMENTAL_INTEGRATIONS" value="false" />
            <ENTRIES>
                <ENTRY IS_ENABLED="true" PARSER="runconfig" IS_EXECUTABLE="false" />
                <ENTRY IS_ENABLED="true" PARSER="env" IS_EXECUTABLE="false" PATH="local/telepresence.env" />
            </ENTRIES>
        </EXTENSION>
        <kind value="DIRECTORY" />
        <package value="github.com/Dynatrace/dynatrace-operator" />
        <directory value="$PROJECT_DIR$/cmd" />
        <filePath value="$PROJECT_DIR$" />
        <method v="2" />
    </configuration>
</component>
```

### VSCode

Add the following to your `launch.json`:

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug Webhook",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/main.go",
            "env": {
                "POD_NAMESPACE": "dynatrace",
                "RUN_LOCAL": "true"
            },
            "envFile": "${workspaceFolder}/local/telepresence.env",
            "args": [
                "webhook-server",
                "--certs-dir=./local/certs/"
            ]
        },
        {
            "name": "Debug Operator",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/main.go",
            "env": {
                "POD_NAMESPACE": "dynatrace",
                "RUN_LOCAL": "true"
            },
            "args": [
                "operator"
            ]
        },
        {
            "name": "Debug CSI driver (server)",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "github.com/Dynatrace/dynatrace-operator",
            "port": 40000,
            "host": "127.0.0.1",
            "apiVersion": 2
        },
        {
            "name": "Debug CSI driver (provisioner)",
            "type": "go",
            "request": "attach",
            "mode": "remote",
            "remotePath": "github.com/Dynatrace/dynatrace-operator",
            "port": 40001,
            "host": "127.0.0.1",
            "apiVersion": 2
        }
    ]
}
```
