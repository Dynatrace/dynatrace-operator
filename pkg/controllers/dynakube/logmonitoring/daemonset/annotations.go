package daemonset

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/latest/dynakube"
	"github.com/Dynatrace/dynatrace-operator/pkg/controllers/dynakube/logmonitoring/configsecret"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/kubernetes/fields/k8ssecuritycontext"
)

const annotationTenantTokenHash = api.InternalFlagPrefix + "tenant-token-hash"

func (r *Reconciler) getAnnotations(dk *dynakube.DynaKube) map[string]string {
	annotations := k8ssecuritycontext.RemoveAppArmorAnnotation(dk.LogMonitoring().Template().Annotations, containerName, initContainerName)

	return configsecret.AddAnnotations(annotations, *dk)
}
