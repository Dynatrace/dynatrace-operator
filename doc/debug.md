# Debugging the operator

This document outlines the distinct debugging requirements for various components, providing detailed instructions for each to ensure effective troubleshooting and development.

## TLDR

### CSI-Driver-server

- **Run CSI driver executables on the node** for file system operations.
- **Makefile commands**:
  - `make debug/build`: Build image with Delve included.
  - `make debug/deploy`: Install image with necessary changes to deployments.
  - `make debug/tunnel`: Open tunnel from local machine to CSI driver pod.
- **IntelliJ setup**: Configure "Go Remote" with `localhost` and port `40000`, and set "On disconnect" to "Leave it running".
- **VSCode:** Add a debug configuration to "Connect to Server" with `127.0.0.1` as the host and `40000` as the port, and set "remotePath" to `github.com/Dynatrace/dynatrace-operator`.

### CSI-driver-provisioner

- **Same steps as CSI-Driver-server**:
  - Same IntelliJ & VSCode debug configuration than previously, but **port changes to** `40001`.

### Operator main code

- **Run operator locally** on your machine.
- **Debugging steps**:
  - Scale down cluster operator: `kubectl -n dynatrace scale --replicas 0 deployment/dynatrace-operator`
  - Run locally with `POD_NAMESPACE=dynatrace RUN_LOCAL=true`.
  - **IntelliJ setup**: Create debug configuration, use `go build`, set directory to `./cmd`, program arguments to `operator`, and set env variables.
  - **VSCode:** Add a new debug configuration using 'Go: Launch package', set the program to `${workspaceFolder}/cmd/main.go`, environment variables to `POD_NAMESPACE=dynatrace RUN_LOCAL=true`, and arguments to `operator`.
  - **In the terminal**: `make debug/operator` to run the operator locally.
- **After debugging**:
  - Scale up cluster operator: `kubectl -n dynatrace scale --replicas 1 deployment/dynatrace-operator`.

### Webhook

- **Run webhook locally** using Telepresence.
- **Steps**:
  - Make sure Telepresence is installed
  - `make debug/build`
  - `make debug/deploy`
  - Install and setup : `make debug/telepresence/install`.
  - All request sent to the service are tunneled to the local port 8443.
  - Start webhook locally in debug mode with env vars from `local/telepresence.env`.
  - **IntelliJ setup**: Install `EnvFile` extension, configure run/debug with `go build`, set directory to `./cmd`, and program arguments to `webhook-server --certs-dir=./local/certs/`.
  - **In VSCode**: Add debug configuration with env file set to `${workspaceFolder}/local/telepresence.env` and args set to `webhook-server --certs-dir=./local/certs/`
  - **In terminal**: `make debug/webhook` to start debugging.
  - When done:
    - Remove tunnel: `make debug/telepresence/stop`
    - Remove all deployed debugging changes: `make install`

### Init-Container

- **Delve debugger injection** is not possible due to port-forwarding limitations.

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
  - `debug/build`: Builds the image with Delve included.
  - `debug/deploy`: Install the image into the cluster. In addition, it
    - Removes the security context from the webhook pod, as Telepresence would copy it and fail.
    - Scales down the webhook pod to 1 replica.
    - Removes the limits from the CSI driver containers, as Delve requires more RAM.
    - Changes the startup command for the CSI driver to use Delve.
  - `debug/tunnel`: Opens a tunnel from your local machine to the CSI driver pod, so your IDE can connect to it.
- After that, just use it in your IDE:
  - IntelliJ:
    - Add a new debug configuration for "Go Remote".
    - Enter `localhost` as the host and `40000` as the port.
    - Set "On disconnect" to "Leave it running".
  - VSCode:
    - Add a debug configuration.
    - Go to "Connect to Server".
    - Enter `127.0.0.1` as the host and `40000` as the port.
    - Change "remotePath" to "github.com/Dynatrace/dynatrace-operator".
- When you are done
  - Remove the debugging patches with `make install`.
  - Turn off the tunnel with `make debug/tunnel/stop`

### CSI-Driver Provisioner

- The same steps as for the server:
- Makefile commands:
  - `debug/build`: Builds the image with Delve included.
  - `debug/deploy`: Install the image into the cluster.
  - `debug/tunnel`: Opens a tunnel from your local machine to the CSI driver pod.
- After that, just use it in your IDE:
  - Same as before, but **change the port** to `40001`.

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
    - `make debug/build`: Not necessary for the webhook, but the next command will also inject the debugger into the CSI driver, so a debug build is required.
    - `make debug/deploy`: Install the image with necessary changes to deployments.
      - This scales down the webhook to 1 replica.
      - It removes the security context from the webhook pod, as Telepresence would copy it and fail.
    - `make debug/telepresence/install`:
      - Install the Telepresence daemon in your cluster.
      - Connect to the cluster.
      - Intercept the request to the webhook.
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
