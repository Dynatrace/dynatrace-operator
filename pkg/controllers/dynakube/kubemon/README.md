# Kubernetes Monitoring Operand (kubemon)

## Why this exists

Today, ActiveGate runs all capabilities — routing, metrics ingest, and kubernetes-monitoring — in a
single StatefulSet. This is a problem because the capabilities have fundamentally different scaling
profiles: routing scales horizontally with ingest traffic, while kubernetes-monitoring runs as a
single memory-heavy instance that scrapes the Kubernetes API. Sharing one pod template means one
`replicas`/`resources` block for both, so users who need independent sizing have to deploy two
separate DynaKubes.

`kubemon` is a new operand that gives kubernetes-monitoring its own StatefulSet, connection secret,
and scaling knobs. Together with the existing Gateway AG (routing, ingest), it forms the split-AG
model: one DynaKube, two independently sized operands. Users no longer need the two-DynaKube
workaround, and the operator can manage each capability's lifecycle separately.

## Trunk-based development approach

- Unfinished kubemon work is merged to `main` early.
- Behavior stays **disabled by default** behind a temporary gate.
- Developers can enable it in test environments to iterate quickly.
- When all missing items are complete, remove the temporary gate.

## Progress

### Done

- Temporary operand gate (`KUBEMON_OPERAND_ENABLED` env var + Helm value)
- kubemon orchestrator with one-condition ownership (`KubernetesMonitoringAvailable`)
- StatefulSet lifecycle and rollout-based availability handling
- Transient vs persistent error mapping with tests
- Self-sufficient connection-info reconciler (kubemon-owned ConfigMap and Secret)
- Runtime wiring in StatefulSet (required env vars, token mount, restart trigger hash)
- Support for core StatefulSet spec propagation (`rollingUpdate`, storage, DNS policy, priority class, termination grace period, ephemeral volume)

### Missing

- KSPM-gated Service, image discovery, registration, custom TLS, custom properties
- Gateway service account and RBAC
- Webhook validation for split-AG mode
- End-to-end tests

## Enable for testing

Use Helm to enable kubemon in dev/test environments:

```zsh
helm upgrade --install dynatrace-operator config/helm/chart/default \
  -n dynatrace --create-namespace \
  --set operator.kubemonOperandEnabled=true
```

The Helm value sets `KUBEMON_OPERAND_ENABLED=true` for operator and webhook pods.
