# Debugging the operator

This document outlines the distinct debugging requirements for various components, providing detailed instructions for each to ensure effective troubleshooting and development.

## TLDR

### CSI-Driver-server

- **Run CSI driver executables on the node** for file system operations.
- **Use Delve debugger**: Include it in the image build, remove `-s` and `-w` flags, allocate more RAM, change container command to use `dlv`, and forward debugger port to localhost.
- **Makefile commands**:
  - `debug/prepare/csi-server`: Prepare for debugging.
  - `debug/tunnel/csi-driver`: Open tunnel for debugging.
  - `debug/remove/csi-server`: Remove debugging patches.
- **IntelliJ setup**: Configure “Go Remote” with `localhost` and port `40000`, and set “On disconnect” to “Leave it running”.
- **VSCode:** Add a debug configuration to "Connect to Server" with `127.0.0.1` as the host and `40000` as the port, and set "remotePath" to `github.com/Dynatrace/dynatrace-operator`.

### CSI-driver-provisioner

- **Same steps as CSI-Driver-server**:
  - `make debug/prepare/csi-provisioner`
  - `debug/tunnel/csi-driver`
  - Same IntelliJ & VSCode debug configuration than previously.

### Operator main code

- **Run operator locally** on your machine.
- **Debugging steps**:
  - Scale down cluster operator: `kubectl -n dynatrace scale --replicas 0 deployment/dynatrace-operator`
  - Run locally with `POD_NAMESPACE=dynatrace RUN_LOCAL=true`.
  - **IntelliJ setup**: Create debug configuration, use `go build`, set directory to `./cmd`, program arguments to `operator`, and set env variables.
  - **VSCode:** Add a new debug configuration using 'Go: Launch package', set the program to `${workspaceFolder}/cmd/main.go`, environment variables to `POD_NAMESPACE=dynatrace RUN_LOCAL=true`, and arguments to `operator`.
  - **In the terminal**: `make debug/operator` to run the operator locally.

### Webhook

- **Run webhook locally** using Telepresence.
- **Steps**:
  - Remove `securityContext` in Helm values.
  - Install operator in cluster.
  - Install Telepresence daemon: `telepresence helm install`.
  - Connect to cluster: `telepresence connect -n dynatrace`.
  - Intercept webhook requests: `telepresence intercept dynatrace-webhook --port 8443 --env-file ./local/telepresence.env`.
  - Start webhook locally in debug mode with env vars from `local/telepresence.env`.
  - **IntelliJ setup**: Install `EnvFile` extension, configure run/debug with `go build`, set directory to `./cmd`, and program arguments to `webhook-server --certs-dir=./local/certs/`.
  - **In VSCode**: Add debug configuration with env file set to `${workspaceFolder}/local/telepresence.env` and args set to `webhook-server --certs-dir=./local/certs/`
  - Stop Telepresence with `telepresence quit`.
  - Uninstall telepresence from cluster: `telepresence helm uninstall`

### Init-Container

- **Delve debugger injection** is not possible due to port-forwarding limitations.

## Longer version

### CSI-Driver Server

- The CSI driver executables must run on the node for all file system-specific operations. This means we need to build the application and remote debug it. Delve, one of the GoLang debuggers, is capable of doing this. However, several changes are required to use its functionality:
  - Include Delve in the image build process, as the debugger must run on the node too.
  - Remove the `extldflags -s` and `-w`:
    - **`-s`**: This flag omits the symbol table and debug information from the binary. The symbol table is used for debugging and profiling, so removing it can significantly reduce the binary size.
    - **`-w`**: This flag omits the DWARF debugging information. DWARF is a widely used, standardized debugging data format. By excluding this information, the binary size is further reduced.
  - Run the relevant container with more RAM, as the debugger uses around 300MB of RAM.
  - Change the container command to use `dlv`, which then starts up the actual operator code.
  - Forward the debugger port to localhost.
- To simplify these steps, I created Makefile commands:
  - `debug/prepare/csi-server`: Patches all files so you have enough RAM and the debugger is included when building.
  - `debug/tunnel/csi-driver`: Opens a tunnel between the debugger and your PC. This must run while debugging.
  - `debug/remove/csi-server`: Removes the patches from before.
- After that, just use it in your IDE:
  - IntelliJ:
    - Add a new debug configuration for “Go Remote”.
    - Enter `localhost` as the host and `40000` as the port.
    - Set “On disconnect” to “Leave it running”.
  - VSCode:
    - Add a debug configuration.
    - Go to “Connect to Server”.
    - Enter `127.0.0.1` as the host and `40000` as the port.
    - Change “remotePath” to “github.com/Dynatrace/dynatrace-operator”.

### CSI-Driver Provisioner

- The same steps as for the server:
  - `make debug/prepare/csi-provisioner`
  - `debug/tunnel/csi-driver`
  - Use the same debug configuration as for the server.

### Operator Main Code

- The operator can run locally on your machine and does not need to run in the cluster.
- For debugging:
  - Scale down your operator running in the cluster: `kubectl -n dynatrace scale --replicas 0 deployment/dynatrace-operator`.
  - Run the operator in debug mode with the following environment variables set: `POD_NAMESPACE=dynatrace RUN_LOCAL=true`.
  - In IntelliJ:
    - Create a new debug configuration.
    - Use `go build`.
    - Set the directory to `./cmd`.
    - Set the program arguments to `operator`.
    - Set the environment variables as stated above.
  - In VSCode:
    - Add a new debug configuration.
    - Use 'Go: Launch package'.
    - Set the following thing:

      ```json
      {
        "name": "Debug operator",
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
        ],
      }
      ```

### Webhook

- In theory, it is possible to run the webhook locally. The problem is that every mutation or validation request is sent from kubelet to the webhook service. So, we need a way to tunnel all requests sent to the service to the local webhook we are running.
- There is a useful tunneling application called Telepresence that does exactly that for us.
  - The following steps require Telepresence, which you can [download here](https://www.telepresence.io/docs/install/client).
  - To install:
    - **Remove the securityContext** in the Helm values YAML file.
      - The problem is that Telepresence would copy it and fail.
    - Ensure that you have installed the operator into the cluster as usual.
    - Install their daemon in your cluster: `telepresence helm install`.
    - Connect to the cluster: `telepresence connect -n dynatrace`.
    - Intercept the request to the webhook: `telepresence intercept dynatrace-webhook --port 8443 --env-file telepresence.env`.
    - Start the webhook locally in debug mode.
      - You have to add the environment variables in `local/telepresence.env`.
      - In IntelliJ:
        - Install the following extension: `EnvFile` (link).
        - Go to run/debug configurations.
        - Add a new one with `go build`.
        - Set the directory to `./cmd`.
        - Set the program arguments to `webhook-server --certs-dir=./local/certs/`.
      - In VSCode
        - Add the following debug configuration:

        ``` json
        {
            "name": "Debug webhook",
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
        ```

      - Set breakpoints and start debugging.
    - When you are done, stop Telepresence with `telepresence quit`.
    - Uninstall from the cluster: `telepresence helm uninstall`.

### Init-Container

- I tried to use the Delve debugger injection, but the problem is that port-forwarding is not possible as long as the pod is not ready. So, we can’t debug there.
