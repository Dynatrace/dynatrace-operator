package kubeobjects

import (
	"context"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ApiRequests[
	BO any,
	O interface {
		*BO
		client.Object
	},
	BL any,
	L interface {
		*BL
		client.ObjectList
	},
] struct {
	context.Context
	client.Reader
	client.Client
	*runtime.Scheme
}

func NewApiRequests[
	BO any,
	O interface {
		*BO
		client.Object
	},
	BL any,
	L interface {
		*BL
		client.ObjectList
	},
](
	context context.Context,
	reader client.Reader,
	client client.Client,
	scheme *runtime.Scheme,
) *ApiRequests[BO, O, BL, L] {
	return &ApiRequests[BO, O, BL, L]{
		Context: context,
		Reader:  reader,
		Client:  client,
		Scheme:  scheme,
	}
}

func (requests *ApiRequests[_, O, _, _]) Create(owner client.Object, toCreate O) error {
	deployed, err := requests.Get(toCreate)
	switch {
	case apierrors.IsNotFound(err):
		err = requests.formOwnership(owner, toCreate)
		if err == nil {
			err = requests.Client.Create(requests.Context, toCreate)
		}
	case err == nil:
		if !requests.underOwnership(owner, deployed) {
			err = errors.Errorf(
				"found colliding %s: %s",
				deployed.GetObjectKind().GroupVersionKind().Kind,
				deployed.GetName())
		}
	}

	return errors.WithStack(err)
}

func (requests *ApiRequests[BO, O, _, _]) Get(toFind O) (O, error) {
	found := O(new(BO))
	err := requests.Reader.Get(
		requests.Context,
		client.ObjectKeyFromObject(toFind),
		found)

	if err != nil {
		found = nil
	}
	return found, err
}

// specify the owner to facilitate the implicite deletion
func (requests *ApiRequests[_, O, _, _]) formOwnership(owner client.Object, minion O) error {
	return errors.WithStack(
		controllerutil.SetControllerReference(
			owner,
			minion,
			requests.Scheme))
}

func (requests *ApiRequests[_, O, _, _]) underOwnership(owner client.Object, minion O) bool {
	for _, scanned := range minion.GetOwnerReferences() {
		if scanned.UID == owner.GetUID() {
			return true
		}
	}
	return false
}

func (requests *ApiRequests[_, O, _, _]) Delete(toDelete O) error {
	_, err := requests.Get(toDelete)
	switch {
	case apierrors.IsNotFound(err):
		err = errors.Errorf(
			"not found deleted %s: %s",
			toDelete.GetObjectKind().GroupVersionKind().Kind,
			toDelete.GetName())
	case err == nil:
		err = requests.Client.Delete(
			requests.Context,
			toDelete)
	}

	return errors.WithStack(err)
}

func (requests *ApiRequests[_, _, BL, L]) List(opts ...client.ListOption) (L, error) {
	toPopulate := L(new(BL))

	err := requests.Reader.List(
		requests.Context,
		toPopulate,
		opts...)
	if err != nil {
		toPopulate = nil
	}

	return toPopulate, err
}

func (requests *ApiRequests[_, O, _, _]) Update(owner client.Object, toUpdate O) error {
	err := requests.formOwnership(owner, toUpdate)
	if err == nil {
		err = requests.Client.Update(
			requests.Context,
			toUpdate)
	}

	return err
}
