#!/bin/bash

export NAMESPACE="${1:-dynatrace}"
export JOB_NAME="dynatrace-cleanup-node-fs"
export MAX_WAIT_SECONDS=300
export WAIT_BEFORE_JOB_DESTRUCTION_SECONDS=0
export CSI_DRIVER_DATA_PATH="/var/lib/kubelet/plugins/csi.oneagent.dynatrace.com"

echo "Using namespace: $NAMESPACE"
kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || {
  echo "Namespace $NAMESPACE does not exist. Creating it..."
  kubectl create namespace "$NAMESPACE"
}

number_of_nodes=$(kubectl get nodes --no-headers | wc -l | tr -d ' ')

echo "Creating cleanup job for $number_of_nodes nodes..."

cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: $NAMESPACE
spec:
  ttlSecondsAfterFinished: 300
  manualSelector: true
  selector:
    matchLabels:
      job-name: ${JOB_NAME}
  parallelism: $number_of_nodes
  completions: $number_of_nodes
  template:
    metadata:
      labels:
        job-name: ${JOB_NAME}
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: job-name
                    operator: In
                    values:
                      - ${JOB_NAME}
              topologyKey: "kubernetes.io/hostname"
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
      restartPolicy: Never
      terminationGracePeriodSeconds: 5
EOF

echo ""
echo "Waiting for all pods to be running (init containers completed)..."

# Wait for all pods to reach Running state
elapsed=0
while [ $elapsed -lt $MAX_WAIT_SECONDS ]; do
  running_count=$(kubectl get pods -n "$NAMESPACE" -l job-name="$JOB_NAME" --field-selector=status.phase=Running --no-headers 2>/dev/null | wc -l | tr -d ' ')
  total_count=$(kubectl get pods -n "$NAMESPACE" -l job-name="$JOB_NAME" --no-headers 2>/dev/null | wc -l | tr -d ' ')
  
  echo "Running pods: $running_count/$number_of_nodes (Total pods: $total_count)"
  
  if [ "$running_count" -eq "$number_of_nodes" ]; then
    echo ""
    echo "✅ All $number_of_nodes pods are running - init containers completed successfully!"
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
kubectl get pods -n "$NAMESPACE" -l job-name="$JOB_NAME"

echo ""
echo "Init container completion status:"
successful_inits=0
failed_inits=0

for pod in $(kubectl get pods -n "$NAMESPACE" -l job-name="$JOB_NAME" --no-headers -o custom-columns=":metadata.name"); do
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
echo "  ✅ Successful cleanups: $successful_inits/$number_of_nodes"
if [ $failed_inits -gt 0 ]; then
  echo "  ❌ Failed cleanups: $failed_inits/$number_of_nodes"
fi

sleep $WAIT_BEFORE_JOB_DESTRUCTION_SECONDS

# Delete the job
echo ""
echo "Deleting cleanup job..."
kubectl delete job "$JOB_NAME" -n "$NAMESPACE"

echo "Restarting CSI driver pods in case they are deployed..."
kubectl -n $NAMESPACE delete pod -l app.kubernetes.io/component=csi-driver,app.kubernetes.io/name=dynatrace-operator

echo ""
echo "Cleanup process completed." 