package namespace

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"

	dynatracev1alpha1 "github.com/Dynatrace/dynatrace-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	utilruntime.Must(scheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(dynatracev1alpha1.AddToScheme(scheme.Scheme))
}

func TestReconcileNamespace(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
		&dynatracev1alpha1.DynaKube{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Spec: dynatracev1alpha1.DynaKubeSpec{
				APIURL: "https://test-url/api",
				CodeModules: dynatracev1alpha1.CodeModulesSpec{
					Enabled: true,
				},
				InfraMonitoring: dynatracev1alpha1.FullStackSpec{
					Enabled: true,
				},
			},
			Status: dynatracev1alpha1.DynaKubeStatus{
				EnvironmentID: "abc12345",
				OneAgent: dynatracev1alpha1.OneAgentStatus{
					Instances: map[string]dynatracev1alpha1.OneAgentInstance{
						"node1": {},
					},
				},
				},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "test-namespace",
				Labels: map[string]string{"oneagent.dynatrace.com/instance": "oneagent"},
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
				UID:  "42",
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "oneagent", Namespace: "dynatrace"},
			Data:       map[string][]byte{"paasToken": []byte("42"), "apiToken": []byte("84")},
		},
	).Build()

	r := ReconcileNamespaces{
		client:    c,
		apiReader: c,
		logger:    zap.New(zap.UseDevMode(true), zap.WriteTo(os.Stdout)),
		namespace: "dynatrace",
	}

	_, err := r.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{Name: "test-namespace"}})
	assert.NoError(t, err)

	var nsSecret corev1.Secret
	require.NoError(t, c.Get(context.TODO(), client.ObjectKey{
		Name:      "dynatrace-oneagent-config",
		Namespace: "test-namespace",
	}, &nsSecret))

	require.Len(t, nsSecret.Data, 1)
	require.Contains(t, nsSecret.Data, "init.sh")
	require.Equal(t, `#!/usr/bin/env bash

set -eu

if [[ -f "/var/lib/dynatrace/oneagent/agent/config/ruxithost.id" ]]; then
	echo "WARNING: full-stack OneAgent has been injected to this container. App-only and full-stack injection can conflict with each other."
fi

api_url="https://test-url/api"
config_dir="/mnt/config"
target_dir="/mnt/oneagent"
paas_token="42"
proxy=""
skip_cert_checks="false"
custom_ca="false"
fail_code=0
cluster_id="42"

declare -A im_nodes
im_nodes=(
	["node1"]="abc12345"
)

set +u
host_tenant="${im_nodes[${K8S_NODE_NAME}]}"
set -u

archive="/mnt/init/tmp.$RANDOM"

if [[ "${FAILURE_POLICY}" == "fail" ]]; then
	fail_code=1
fi

if [[ "${INSTALLER_URL}" != "" ]] || [[ "${USE_IMMUTABLE_IMAGE}" != "true" ]]; then
	curl_params=(
		"--silent"
		"--output" "${archive}"
	)

	if [[ "${INSTALLER_URL}" != "" ]]; then
		curl_params+=("${INSTALLER_URL}")
	else
		curl_params+=(
			"${api_url}/v1/deployment/installer/agent/unix/paas/latest?flavor=${FLAVOR}&include=${TECHNOLOGIES}&bitness=64"
			"--header" "Authorization: Api-Token ${paas_token}"
		)
	fi

	if [[ "${skip_cert_checks}" == "true" ]]; then
		curl_params+=("--insecure")
	fi

	if [[ "${custom_ca}" == "true" ]]; then
		curl_params+=("--cacert" "${config_dir}/ca.pem")
	fi

	if [[ "${proxy}" != "" ]]; then
		curl_params+=("--proxy" "${proxy}")
	fi

	echo "Downloading OneAgent package..."
	if ! curl "${curl_params[@]}"; then
		echo "Failed to download the OneAgent package."
		exit "${fail_code}"
	fi

	echo "Unpacking OneAgent package..."
	if ! unzip -o -d "${target_dir}" "${archive}"; then
		echo "Failed to unpack the OneAgent package."
		mv "${archive}" "${target_dir}/package.zip"
		exit "${fail_code}"
	fi
else
	echo "Copy OneAgent package..."
	if ! cp -r "/opt/dynatrace/oneagent/." "${target_dir}"; then
		echo "Failed to copy the OneAgent package."
		exit "${fail_code}"
	fi
fi

echo "Configuring OneAgent..."
echo -n "${INSTALLPATH}/agent/lib64/liboneagentproc.so" >> "${target_dir}/ld.so.preload"

for i in $(seq 1 $CONTAINERS_COUNT)
do
	container_name_var="CONTAINER_${i}_NAME"
	container_image_var="CONTAINER_${i}_IMAGE"

	container_name="${!container_name_var}"
	container_image="${!container_image_var}"

	container_conf_file="${target_dir}/container_${container_name}.conf"

	echo "Writing ${container_conf_file} file..."
	echo "[container]
containerName ${container_name}
imageName ${container_image}
k8s_fullpodname ${K8S_PODNAME}
k8s_poduid ${K8S_PODUID}
k8s_containername ${container_name}
k8s_basepodname ${K8S_BASEPODNAME}
k8s_namespace ${K8S_NAMESPACE}">>${container_conf_file}

	if [[ ! -z "${host_tenant}" ]]; then		
		if [[ "abc12345" == "${host_tenant}" ]]; then
			echo "k8s_node_name ${K8S_NODE_NAME}
k8s_cluster_id ${cluster_id}">>${container_conf_file}
		fi

	echo "
[host]
tenant ${host_tenant}">>${container_conf_file}
	fi
done
`, string(nsSecret.Data["init.sh"]))
}

