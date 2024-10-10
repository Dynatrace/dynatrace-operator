/*
Copyright 2021 Dynatrace LLC.

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

package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/exp"
	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

var (
	log = logd.Get().WithName("dynakube-api")
)

func (dk *DynaKube) FF() *exp.FeatureFlags {
	return exp.NewFeatureFlags(dk.Annotations)
}

func (dk *DynaKube) GetOneAgentInitialConnectRetry() int {
	return dk.FF().GetOneAgentInitialConnectRetry(dk.Spec.EnableIstio)
}

func (dk *DynaKube) GetIgnoredNamespaces() []string {
	return dk.FF().GetIgnoredNamespaces(dk)
}
