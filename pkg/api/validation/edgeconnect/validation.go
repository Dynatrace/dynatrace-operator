package validation

import (
	"context"
	"fmt"

	v1alpha1 "github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha1/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/v1alpha2/edgeconnect"
	"github.com/Dynatrace/dynatrace-operator/pkg/api/validation"
	"github.com/Dynatrace/dynatrace-operator/pkg/util/installconfig"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type validator[T runtime.Object] struct {
	*validatorClient
}

type validatorClient struct {
	apiReader client.Reader
	cfg       *rest.Config
	modules   installconfig.Modules
}

type validatorFunc func(ctx context.Context, dv *validatorClient, ec *edgeconnect.EdgeConnect) string

var validatorErrorFuncs = []validatorFunc{
	isModuleDisabled,
	checkAPIServerProtocolNotSet,
	isAllowedSuffixAPIServer,
	nameTooLong,
	checkHostPatternsValue,
	isInvalidServiceName,
	automationRequiresProvisionerValidation,
}

func newValidator[T runtime.Object](vc *validatorClient) *validator[T] {
	return &validator[T]{validatorClient: vc}
}

func newClient(apiReader client.Reader, cfg *rest.Config) *validatorClient {
	return &validatorClient{
		apiReader: apiReader,
		cfg:       cfg,
		modules:   installconfig.GetModules()}
}

func (v *validator[T]) ValidateCreate(ctx context.Context, obj T) (_ admission.Warnings, err error) {
	ec, err := getEdgeConnect(obj)
	if err != nil {
		return
	}

	validationErrors := v.runValidators(ctx, validatorErrorFuncs, ec)

	if len(validationErrors) > 0 {
		err = errors.New(validation.SumErrors(validationErrors, "EdgeConnect"))
	}

	return
}

func (v *validator[T]) ValidateUpdate(ctx context.Context, _, newObj T) (warnings admission.Warnings, err error) {
	ec, err := getEdgeConnect(newObj)
	if err != nil {
		return
	}

	validationErrors := v.runValidators(ctx, validatorErrorFuncs, ec)

	if len(validationErrors) > 0 {
		err = errors.New(validation.SumErrors(validationErrors, "EdgeConnect"))
	}

	return
}

func (v *validator[T]) ValidateDelete(_ context.Context, _ T) (warnings admission.Warnings, err error) {
	return nil, nil
}

func (v *validator[T]) runValidators(ctx context.Context, validators []validatorFunc, ec *edgeconnect.EdgeConnect) []string {
	results := []string{}

	for _, validate := range validators {
		if errMsg := validate(ctx, v.validatorClient, ec); errMsg != "" {
			results = append(results, errMsg)
		}
	}

	return results
}

func getEdgeConnect(obj runtime.Object) (ec *edgeconnect.EdgeConnect, err error) {
	ec = &edgeconnect.EdgeConnect{}

	switch v := obj.(type) {
	case *edgeconnect.EdgeConnect:
		ec = v
	case *v1alpha1.EdgeConnect:
		err = v.ConvertTo(ec)
	default:
		if gvk := obj.GetObjectKind().GroupVersionKind(); !gvk.Empty() {
			return nil, fmt.Errorf("unknown object %s", gvk)
		}

		return nil, fmt.Errorf("unknown object %T", obj)
	}

	return
}
