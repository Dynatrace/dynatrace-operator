package hasher

import (
	"encoding/json"
	"hash/fnv"
	"reflect"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AnnotationHash = api.InternalFlagPrefix + "template-hash"

func GenerateHash(ds any) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", errors.WithStack(err)
	}

	hasher := fnv.New32()

	_, err = hasher.Write(data)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func IsDifferent(a, b any) (bool, error) {
	hashA, err := GenerateHash(a)
	if err != nil {
		return false, err
	}

	hashB, err := GenerateHash(b)
	if err != nil {
		return false, err
	}

	return hashA != hashB, nil
}

func IsAnnotationDifferent(a, b metav1.Object) bool {
	return getHash(a) != getHash(b)
}

func getHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[AnnotationHash]
	}

	return ""
}

func AddAnnotation(object metav1.Object) error {
	if object == nil || reflect.ValueOf(object).IsNil() {
		return errors.New("nil objects can't have a hash annotation")
	}

	annotations := object.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	objectHash, err := GenerateHash(object)
	if err != nil {
		return err
	}

	annotations[AnnotationHash] = objectHash
	object.SetAnnotations(annotations)

	return nil
}
