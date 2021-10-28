# Changelog

### [v0.3.0](https://github.com/Dynatrace/dynatrace-operator/releases/tag/v0.3.0)

#### New Features

* implemented validation webhook for dynakube objects [#210](https://github.com/Dynatrace/dynatrace-operator/pull/210)

#### Bug fixes

* Change tenant primary key to Dynakube [#289](https://github.com/Dynatrace/dynatrace-operator/pull/289)

#### Code Improvements

* Change ImagePullPolicy to IfNotPresent for init container [#299](https://github.com/Dynatrace/dynatrace-operator/pull/299)
* Add oneagent installer environment variables when needed [#296](https://github.com/Dynatrace/dynatrace-operator/pull/296)
* Ignore base kubernetes/openshift namespaces by default, tolerations for csi via feature-flag [#294](https://github.com/Dynatrace/dynatrace-operator/pull/294)
* Provides the hostGroup to the special agents [#284](https://github.com/Dynatrace/dynatrace-operator/pull/284)
* Add unit test for conversion webhook [#280](https://github.com/Dynatrace/dynatrace-operator/pull/280)
* Add log message for monitored namespaces for cloudNative and appOnly [#276](https://github.com/Dynatrace/dynatrace-operator/pull/276)
* Add resources to all csi driver containers [#262](https://github.com/Dynatrace/dynatrace-operator/pull/262)
* Add certificate controller enhancement [#243](https://github.com/Dynatrace/dynatrace-operator/pull/243)

#### Documentation Updates

* sample CRs adaption - update to new CRD [#300](https://github.com/Dynatrace/dynatrace-operator/pull/300)

### [v0.2.2](https://github.com/Dynatrace/dynatrace-operator/releases/tag/v0.2.2)

#### Bug fixes

* Fixed a bug where the proxy setting was not properly passed when using immutable images [#213](https://github.com/Dynatrace/dynatrace-operator/pull/213)
* Fixed a bug where the proxy setting was expected in the wrong field in the secret when provided via proxy.valueFrom [#218](https://github.com/Dynatrace/dynatrace-operator/pull/218)

#### Other changes

* Removed PodSecurityPolicies since they got removed with Kubernetes 1.22 [#215](https://github.com/Dynatrace/dynatrace-operator/pull/215)
* Updated the apiVersion of the CRD from v1beta1 to v1 since v1beta1 got removed with Kubernetes 1.22 [#216](https://github.com/Dynatrace/dynatrace-operator/pull/216)

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
