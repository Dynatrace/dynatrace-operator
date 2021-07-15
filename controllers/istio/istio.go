package istio

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var (
	istioGVRName = "networking.istio.io"

	// VirtualServiceGVK => definition of virtual service GVK for oneagent
	VirtualServiceGVK = schema.GroupVersionKind{
		Group:   istioGVRName,
		Version: "v1alpha3",
		Kind:    "VirtualService",
	}

	// ServiceEntryGVK => definition of virtual service GVK for oneagent
	ServiceEntryGVK = schema.GroupVersionKind{
		Group:   istioGVRName,
		Version: "v1alpha3",
		Kind:    "ServiceEntry",
	}
)

// CheckIstioEnabled checks if Istio is installed
func CheckIstioEnabled(cfg *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}
	apiGroupList, err := client.ServerGroups()
	if err != nil {
		return false, err
	}

	for _, apiGroup := range apiGroupList.Groups {
		if apiGroup.Name == istioGVRName {
			return true, nil
		}
	}
	return false, nil
}

// BuildNameForEndpoint returns a name to be used as a base to identify Istio objects.
func buildNameForEndpoint(name string, protocol string, host string, port uint32) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%s-%d", name, protocol, host, port)))
	return hex.EncodeToString(sum[:])
}

func buildObjectMeta(name string, namespace string) v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      name,
		Namespace: namespace,
	}
}

func mapErrorToObjectProbeResult(err error) (probeResult, error) {
	if err != nil {
		if errors.IsNotFound(err) {
			return probeObjectNotFound, err
		} else if meta.IsNoMatchError(err) {
			return probeTypeNotFound, err
		}

		return probeUnknown, err
	}

	return probeObjectFound, nil
}

func buildIstioLabels(name, role string) map[string]string {
	return map[string]string{
		"dynatrace":            "oneagent",
		"oneagent":             name,
		"dynatrace-istio-role": role,
	}
}

func isIp(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	return false
}
