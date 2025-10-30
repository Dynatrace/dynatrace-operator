#!/bin/bash

export NAMESPACE="${1:-dynatrace}"
export DAEMONSET_NAME="dynatrace-cleanup-node-fs"
export MAX_WAIT_SECONDS=300
export WAIT_BEFORE_DAEMONSET_DESTRUCTION_SECONDS=0
export CSI_DRIVER_DATA_PATH="/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com"

echo "Using namespace: $NAMESPACE"
kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || {
  echo "Namespace $NAMESPACE does not exist. Creating it..."
  kubectl create namespace "$NAMESPACE"
}

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
      initContainers:
      - name: cleanup-init
        image: registry.access.redhat.com/ubi9-micro:9.6
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
                    rm -rf "\$dir"
                    echo "Removed OneAgent directory: \$dir"
                else
                    echo "OneAgent directory not found (skipping): \$dir"
                fi
            done

            echo 'Removing CSI driver directory...';
            if rm -rf /mnt/root${CSI_DRIVER_DATA_PATH} 2>&1; then
                echo 'CSI driver directory removed successfully.';
            else
                echo 'WARNING: Failed to remove CSI driver directory (may not exist).';
            fi
            
            if [ \$exit_code -eq 0 ]; then
                echo 'SUCCESS: Node filesystem cleanup completed successfully.';
            else
                echo 'FAILURE: Node filesystem cleanup completed with errors.';
            fi
            
            # Always exit 0 to prevent restarts - we'll check logs for SUCCESS/FAILURE
            exit 0
        volumeMounts:
        - name: host-root
          mountPath: /mnt/root
      containers:
      - name: main
        image: registry.access.redhat.com/ubi9-micro:9.6
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
  
  echo "DaemonSet status: Ready: $ready/$desired (Current scheduled: $current)"
  
  if [ "$ready" -eq "$desired" ] && [ "$desired" -gt "0" ]; then
    echo ""
    echo "✅ All $desired DaemonSet pods are ready - cleanup completed successfully!"
    break
  fi
  
  sleep 2
  elapsed=$((elapsed + 2))
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
    # Check logs for SUCCESS or FAILURE marker
    logs=$(kubectl logs "$pod" -n "$NAMESPACE" -c cleanup-init 2>/dev/null || echo "")
    
    if echo "$logs" | grep -q "SUCCESS: Node filesystem cleanup completed successfully"; then
      successful_inits=$((successful_inits + 1))
    elif echo "$logs" | grep -q "FAILURE: Node filesystem cleanup completed with errors"; then
      failed_inits=$((failed_inits + 1))
      echo "  ❌ $pod - cleanup failed (check logs: kubectl logs $pod -n $NAMESPACE -c cleanup-init)"
    else
      # Init container terminated but no clear status
      echo "  ⚠️  $pod - unknown status"
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

echo ""
echo "Restarting CSI driver pods in case they are deployed..."
kubectl -n $NAMESPACE delete pod -l app.kubernetes.io/component=csi-driver,app.kubernetes.io/name=dynatrace-operator

echo ""
echo "Cleanup process completed." 