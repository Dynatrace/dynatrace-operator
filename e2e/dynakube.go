// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PhaseWait interface {
	WaitForPhase(dynatracev1alpha1.DynaKubePhaseType) error
}

type waitConfiguration struct {
	clt           client.Client
	maxWaitCycles int
	namespace     string
	name          string
	t             *testing.T
}

func NewOneAgentWaitConfiguration(t *testing.T, clt client.Client, maxWaitCycles int, namesapce string, name string) PhaseWait {
	return &waitConfiguration{
		clt:           clt,
		maxWaitCycles: maxWaitCycles,
		namespace:     namesapce,
		name:          name,
		t:             t,
	}
}

func (waitConfig *waitConfiguration) WaitForPhase(phase dynatracev1alpha1.DynaKubePhaseType) error {
	instance := dynatracev1alpha1.DynaKube{}
	iteration := 0
	for iteration < waitConfig.maxWaitCycles {
		err := waitConfig.clt.Get(context.TODO(),
			client.ObjectKey{Namespace: waitConfig.namespace, Name: waitConfig.name},
			&instance)
		assert.NoError(waitConfig.t, err)

		if instance.Status.Phase == phase {
			return nil
		}
		time.Sleep(30 * time.Second)
		iteration++
	}
	return errors.Errorf("oneagent did not reach desired phase '%s'", phase)
}
