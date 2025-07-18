package dynakube

import (
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1beta4/dynakube/oneagent"
)

func (dk *DynaKube) OneAgent() *oneagent.OneAgent {
	oa := oneagent.NewOneAgent(
		&dk.Spec.OneAgent,
		&dk.Status.OneAgent,
		&dk.Status.CodeModules,
		dk.Name,
		dk.APIURLHost(),
		dk.FF().IsOneAgentPrivileged(),
		dk.FF().SkipOneAgentLivenessProbe(),
		dk.FF().IsNodeImagePull(),
	)

	return oa
}
