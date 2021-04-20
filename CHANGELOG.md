# Changelog

### Future

#### Bug fixes
* Detection of OneAgent upgrades doesn't depend on individual OneAgent versions in hosts, but rather a new DaemonSet rollout is applied, which should bring more stable upgrades ([#122](https://github.com/Dynatrace/dynatrace-operator/pull/122))

### [v0.2.1](https://github.com/Dynatrace/dynatrace-operator/releases/tag/v0.2.1)

#### Bug fixes

* Fixed a bug where setting the resources for routing was not
  possible ([#114](https://github.com/Dynatrace/dynatrace-operator/pull/114))

### [v0.2.0](https://github.com/Dynatrace/dynatrace-operator/releases/tag/v0.2.0)

#### Features

* classicFullStack - The Dynatrace Operator now supports rolling out a fullstack agent
* routing - The Dynatrace Operator now supports rolling out a containerized ActiveGate to route the agent traffic

### [v0.1.0](https://github.com/Dynatrace/dynatrace-operator/releases/tag/v0.1.0)

* Initial release of the Dynatrace Operator
