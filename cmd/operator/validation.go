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
	validationhook "github.com/Dynatrace/dynatrace-operator/webhook/validation"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startValidationServer(mgr manager.Manager) (manager.Manager, func(), error) {
	cleanUp := func() {}
	if err := validationhook.AddDynakubeValidationWebhookToManager(mgr); err != nil {
		return nil, cleanUp, err
	}

	return mgr, cleanUp, nil
}
