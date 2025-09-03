package hasher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash/fnv"
	"reflect"
	"strconv"

	"github.com/Dynatrace/dynatrace-operator/pkg/api"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AnnotationHash = api.InternalFlagPrefix + "template-hash"

// GenerateHash creates a hash from the provided input.
// This hash is meant to be used for simplifying detecting differences between 2 objects.
// Uses FNV-1 hashing, should be used for hashing not sensitive data.
func GenerateHash(input any) (string, error) {
	data, err := json.Marshal(input)
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

// GenerateSecureHash creates a hash from the provided input.
// This hash is meant to be used for simplifying detecting differences between 2 values.
// Uses SHA256 hashing, can be used for hashing sensitive data.
func GenerateSecureHash(input any) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", errors.WithStack(err)
	}

	hasher := sha256.New()
	hasher.Write(data)

	return hex.EncodeToString(hasher.Sum(nil))[:16], nil
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

	if _, ok := annotations[AnnotationHash]; !ok {
		objectHash, err := GenerateHash(object)
		if err != nil {
			return err
		}

		annotations[AnnotationHash] = objectHash
		object.SetAnnotations(annotations)
	}

	return nil
}
