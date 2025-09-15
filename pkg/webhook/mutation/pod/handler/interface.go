package handler

import dtwebhook "github.com/Dynatrace/dynatrace-operator/pkg/webhook/mutation/pod/mutator"

type Handler interface {
	Handle(mutationRequest *dtwebhook.MutationRequest) error
}
