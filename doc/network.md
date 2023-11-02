# Operator network

- [Kubernetes network policies](#kubernetes-network-policies)
- [Used ports](#used-ports)
  - [Ingress](#ingress)
    - [dynatrace-operator](#dynatrace-operator)
    - [activegate](#activegate)
  - [Egress](#egress)
    - [dynatrace-operator](#dynatrace-operator-1)
    - [kube-system](#kube-system)

## Kubernetes network policies

All network policies in compliance with the operator can be found in `assets/calico`.

- [activegate-policy.yaml](../assets/calico/activegate-policy.yaml)
- [activegate-policy-external-only.yaml](../assets/calico/agent-policy-external-only.yaml)
- [agent-policy.yaml](../assets/calico/agent-policy.yaml)
- [dynatrace-policies.yaml](../assets/calico/dynatrace-policies.yaml)

## Used ports

### Ingress

- `TCP 80`: Default HTTP
- `TCP 443`: Default HTTPS

#### dynatrace-operator

- `TCP 8383`: Webhook server metrics
- `TCP 8384`: Webhook server validation
- `TCP 8443`: Webhook server port
- `TCP 8080`: CSI-Driver server metrics
- `TCP 10080`: CSI-Driver probe

#### activegate

- `TCP 9999`: HTTPS container port
- `TCP 9998`: HTTP container port

#### csi-driver

- `TCP 10090`: CSI-Driver provisioner

### Egress

- `TCP 80`: Default HTTP
- `TCP 443`: Default HTTPS

#### dynatrace-operator

- `TCP 8383`: Webhook server metrics
- `TCP 8384`: Webhook server validation
- `TCP 8443`: Webhook server port
- `TCP 8080`: CSI-Driver server metrics
- `TCP 10080`: CSI-Driver probe

#### kube-system

- `TCP 53`: Allow DNS lookup
- `UPD 53`: Allow DNS lookup
