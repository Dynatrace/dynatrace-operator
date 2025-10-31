#!/bin/bash

# DESCRIPTION:
#   This script deploys a DaemonSet that runs on all Linux nodes (amd64/arm64) in the cluster
#   to clean up Dynatrace OneAgent installations and CSI driver artifacts. It uses an init
#   container to perform the cleanup, ensuring each node is cleaned exactly once without restarts.
#
# USAGE:
#   ./cleanup-node-fs.sh [NAMESPACE]

export NAMESPACE="${1:-dynatrace}"
export DAEMONSET_NAME="dynatrace-cleanup-node-fs"
export MAX_WAIT_SECONDS=600
export WAIT_BEFORE_DAEMONSET_DESTRUCTION_SECONDS=0
export KUBELET_PATH="/var/lib/kubelet"
# renovate: datasource=docker depName=registry.access.redhat.com/ubi9-micro
export UBI_MICRO_IMAGE="registry.access.redhat.com/ubi9-micro:9.6-1760515026@sha256:aff810919642215e15c993b9bbc110dbcc446608730ad24499dafd9df7a8f8f4"

echo "Using namespace: $NAMESPACE"
namespace_created=false
kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || {
  echo "Namespace $NAMESPACE does not exist. Creating it..."
  kubectl create namespace "$NAMESPACE"
  namespace_created=true
}

running_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l | tr -d ' ')
if [ "$running_pods" -gt 0 ]; then
  echo ""
  echo "⚠️  WARNING: Found $running_pods running pod(s) in namespace $NAMESPACE"
  echo ""
  echo "Make sure that no DynaKube is deployed and all monitored pods were restarted before running this cleanup script."
  echo ""
  
  if [ "$SKIP_RUNNING_PODS_WARNING" = true ]; then
    echo "Skipping running pods warning. Continuing with cleanup..."
  else
    read -p "Do you want to continue anyway? (yes/no): " response
    case "$response" in
      [yY][eE][sS]|[yY])
        echo "Continuing with cleanup..."
        ;;
      *)
        echo "Cleanup cancelled."
        exit 0
        ;;
    esac
  fi
fi

echo ""
echo "Creating cleanup DaemonSet..."

cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: ${DAEMONSET_NAME}
  namespace: $NAMESPACE
spec:
  selector:
    matchLabels:
      app: ${DAEMONSET_NAME}
  template:
    metadata:
      labels:
        app: ${DAEMONSET_NAME}
    spec:
      nodeSelector:
        kubernetes.io/os: linux
      tolerations:
      - key: ToBeDeletedByClusterAutoscaler
        operator: Exists
        effect: NoSchedule
      - effect: NoSchedule
        key: node-role.kubernetes.io/master
        operator: Exists
      - effect: NoSchedule
        key: node-role.kubernetes.io/control-plane
        operator: Exists
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
            - matchExpressions:
              - key: kubernetes.io/arch
                operator: In
                values:
                - amd64
                - arm64
                - ppc64le
                - s390x
      initContainers:
      - name: cleanup-init
        image: ${UBI_MICRO_IMAGE}
        command: ["/bin/sh", "-c"]
        args:
          - |
            set +e  # Don't exit on errors, we'll handle them ourselves
            exit_code=0
            
            echo 'Starting node filesystem cleanup...';
            
            if [ -f /mnt/root/opt/dynatrace/oneagent/agent/uninstall.sh ]; then
                echo 'Executing OneAgent uninstall script...';
                chroot /mnt/root /opt/dynatrace/oneagent/agent/uninstall.sh
                rc=\$?
                if [ \$rc -ne 0 ]; then
                    echo "ERROR: OneAgent uninstall script failed with exit code \$rc";
                    exit_code=1
                else
                    echo 'OneAgent uninstall script completed successfully.';
                fi
            else
                echo 'OneAgent uninstall script not found, skipping...';
            fi;

            echo 'Removing OneAgent files if they exist...';
            directories=(
                /mnt/root/var/lib/dynatrace
                /mnt/root/opt/dynatrace
                /mnt/root/var/log/dynatrace
            )

            for dir in "\${directories[@]}"; do
                if [ -d "\$dir" ]; then
                    if rm -rf "\$dir" 2>&1; then
                        echo "Removed OneAgent directory: \$dir"
                    else
                        echo "ERROR: Failed to remove OneAgent directory: \$dir"
                        exit_code=1
                    fi
                else
                    echo "OneAgent directory not found (skipping): \$dir"
                fi
            done

            echo 'Removing CSI driver directory...';
            if rm -rf /mnt/root${KUBELET_PATH}/plugins/csi.oneagent.dynatrace.com/data 2>&1; then
                echo 'CSI driver directory removed successfully.';
            else
                echo 'ERROR: Failed to remove CSI driver directory.';
                exit_code=1
            fi
            
            if [ \$exit_code -eq 0 ]; then
                echo 'SUCCESS: Node filesystem cleanup completed successfully.';
                echo 'SUCCESS' > /dev/termination-log
            else
                echo 'FAILURE: Node filesystem cleanup completed with errors.';
                echo 'FAILURE' > /dev/termination-log
            fi
            
            # Always exit 0 to prevent restarts - we'll check termination log for SUCCESS/FAILURE
            exit 0
        volumeMounts:
        - name: host-root
          mountPath: /mnt/root
        resources:
          requests:
            cpu: 50m
          limits:
            cpu: 100m
        securityContext:
          runAsUser: 0
          allowPrivilegeEscalation: true
          privileged: true
      containers:
      - name: main
        image: ${UBI_MICRO_IMAGE}
        command: ["/bin/sh", "-c"]
        args:
          - |
            # Keep the pod running so the scheduler does not schedule another job on this node
            echo 'Cleanup completed, giving other pods time to complete...';
            sleep infinity;
      volumes:
        - hostPath:
            path: /
            type: ""
          name: host-root
      restartPolicy: Always
      terminationGracePeriodSeconds: 5
EOF

echo ""
echo "Waiting for all cleanup pods to be finished..."

# Wait for all pods to reach Running state
elapsed=0
while [ $elapsed -lt $MAX_WAIT_SECONDS ]; do
  desired=$(kubectl get daemonset "$DAEMONSET_NAME" -n "$NAMESPACE" -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
  ready=$(kubectl get daemonset "$DAEMONSET_NAME" -n "$NAMESPACE" -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
  current=$(kubectl get daemonset "$DAEMONSET_NAME" -n "$NAMESPACE" -o jsonpath='{.status.currentNumberScheduled}' 2>/dev/null || echo "0")
  
  printf "\rDaemonSet status: Ready: %s/%s (Current scheduled: %s) - %ss elapsed" "$ready" "$desired" "$current" "$elapsed"
  
  if [ "$ready" -eq "$desired" ] && [ "$desired" -gt "0" ]; then
    echo ""
    echo "✅ All $desired DaemonSet pods are ready - cleanup completed successfully!"
    break
  fi
  
  sleep 5
  elapsed=$((elapsed + 5))
done

if [ $elapsed -ge $MAX_WAIT_SECONDS ]; then
  echo ""
  echo "⚠️  Timeout waiting for all pods to be running"
fi

# Get detailed status
echo ""
echo "Pod status summary:"
kubectl get pods -n "$NAMESPACE" -l app="$DAEMONSET_NAME"

echo ""
echo "Init container completion status:"
successful_inits=0
failed_inits=0
total_pods=0

for pod in $(kubectl get pods -n "$NAMESPACE" -l app="$DAEMONSET_NAME" --no-headers -o custom-columns=":metadata.name"); do
  total_pods=$((total_pods + 1))
  # Check if init container completed (it always exits 0 to prevent restarts)
  init_status=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.initContainerStatuses[0].state}' 2>/dev/null)
  
  if echo "$init_status" | grep -q "terminated"; then
    # Check termination message for SUCCESS or FAILURE marker
    termination_message=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.initContainerStatuses[0].state.terminated.message}' 2>/dev/null)
    
    if [ "$termination_message" = "SUCCESS" ]; then
      successful_inits=$((successful_inits + 1))
    elif [ "$termination_message" = "FAILURE" ]; then
      failed_inits=$((failed_inits + 1))
      echo "  ❌ $pod - cleanup failed (check logs: kubectl logs $pod -n $NAMESPACE -c cleanup-init)"
    else
      # Init container terminated but no clear status
      echo "  ⚠️  $pod - unknown status (termination message: '$termination_message')"
    fi
  else
    echo "  ⏳ $pod - init container still running or pending"
  fi
done

echo ""
echo "📊 Cleanup Results:"
echo "  ✅ Successful cleanups: $successful_inits/$total_pods"
if [ $failed_inits -gt 0 ]; then
  echo "  ❌ Failed cleanups: $failed_inits/$total_pods"
fi

sleep $WAIT_BEFORE_DAEMONSET_DESTRUCTION_SECONDS

# Only delete the DaemonSet if all cleanups were successful
if [ $failed_inits -eq 0 ] && [ $successful_inits -eq $total_pods ] && [ $total_pods -gt 0 ]; then
  echo ""
  echo "✅ All cleanups successful. Deleting cleanup DaemonSet..."
  kubectl delete daemonset "$DAEMONSET_NAME" -n "$NAMESPACE"
else
  echo ""
  echo "⚠️  Some cleanups failed or are incomplete. Keeping DaemonSet for investigation."
  echo "    To view logs: kubectl logs -n $NAMESPACE -l app=$DAEMONSET_NAME -c cleanup-init"
  echo "    To delete manually: kubectl delete daemonset $DAEMONSET_NAME -n $NAMESPACE"
fi

if [ "$namespace_created" = true ]; then
  echo ""
  echo "Deleting namespace $NAMESPACE (created by this script)..."
  kubectl delete namespace "$NAMESPACE" --ignore-not-found
fi

echo ""
echo "Cleanup process completed." 
