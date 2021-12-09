package kubeobjects

import (
	"encoding/json"
	"hash/fnv"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const AnnotationHash = "internal.operator.dynatrace.com/template-hash"

func GenerateHash(ds interface{}) (string, error) {
	data, err := json.Marshal(ds)
	if err != nil {
		return "", err
	}

	hasher := fnv.New32()
	_, err = hasher.Write(data)
	if err != nil {
		return "", err
	}

	return strconv.FormatUint(uint64(hasher.Sum32()), 10), nil
}

func HasChanged(a, b metav1.Object) bool {
	return getHash(a) != getHash(b)
}

func getHash(a metav1.Object) string {
	if annotations := a.GetAnnotations(); annotations != nil {
		return annotations[AnnotationHash]
	}
	return ""
}
