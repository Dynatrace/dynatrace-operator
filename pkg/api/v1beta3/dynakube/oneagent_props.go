package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta3/dynakube/oneagent"
)

func (dk *DynaKube) OneAgent() *oneagent.OneAgent {
	oa := oneagent.NewOneAgent(
		&dk.Spec.OneAgent,
		&dk.Status.OneAgent,
		&dk.Status.CodeModules,
		dk.Name,
		dk.ApiUrlHost(),
		dk.FF().IsOneAgentPrivileged(),
		dk.FF().SkipOneAgentLivenessProbe())

	return oa
}
