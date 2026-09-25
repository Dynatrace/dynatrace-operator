#!/bin/bash

# DESCRIPTION:
#   This script deploys a DaemonSet that runs on all Linux nodes (amd64/arm64/ppc64le/s390x)
#   in the cluster to clean up Dynatrace OneAgent installations and/or CSI driver artifacts.
#   It uses an init container to perform the cleanup, ensuring each node is cleaned exactly
#   once without restarts.
#
#   What gets cleaned depends on the selected mode:
#     classic - artifacts of a classicFullStack (host-installed) OneAgent:
#                 /opt/dynatrace, /var/lib/dynatrace, /var/log/dynatrace
#               and runs /opt/dynatrace/oneagent/agent/uninstall.sh first, if present.
#               Additionally removes system-level residue (even when uninstall.sh is
#               missing or failed): oneagent entries in /etc/ld.so.preload, oneagent
#               systemd units, and still-running host processes started from dynatrace
#               paths. Containerized agents (CSI-based installs) are never touched -
#               only processes running directly on the host are killed.
#               NOTE: these paths are only written when the CSI driver is NOT in use,
#               so classic cleanup is safe to run alongside a working CSI-based
#               (cloudNativeFullStack/hostMonitoring) installation.
#     csi     - artifacts of CSI-based deployments:
#                 <kubelet>/plugins/csi.oneagent.dynatrace.com/data
#                 /var/opt/dynatrace (default storageHostPath of non-CSI readonly installs)
#               Can be scoped to a subdirectory of the CSI data dir via --csi-subpath
#               (e.g. a tenant id, or _dynakubes/<name>) for surgical troubleshooting.
#     all     - both of the above.
#
#   Before the DaemonSet is removed, the init container logs and pod manifests of every
#   cleanup pod are saved to a local report directory, and each node reports a per-path
#   result (removed/missing/failed/leftover) via its termination message.
#
# USAGE:
#   ./cleanup-node-fs.sh [NAMESPACE] [options]
#
#   Without --mode the script interactively asks what to clean up.
#
# OPTIONS:
#   -n, --namespace NS    namespace to deploy the cleanup DaemonSet into (default: dynatrace;
#                         the positional NAMESPACE argument is kept for backwards compatibility)
#   -m, --mode MODE       what to clean: classic | csi | all (skips the interactive menu)
#       --csi-subpath P   only remove <csi-data-dir>/P instead of the whole CSI data dir
#                         (implies: /var/opt/dynatrace is left untouched)
#       --extra-dir DIR   additional host directory to remove, repeatable
#                         (e.g. a custom spec.oneAgent...storageHostPath)
#       --kubelet-path P  kubelet root on the nodes (default: /var/lib/kubelet;
#                         adjust for k0s/microk8s/RKE2 style distros)
#       --report-dir DIR  where to store per-node logs and pod manifests
#                         (default: ./dynatrace-cleanup-report-<timestamp>)
#   -y, --yes             non-interactive: skip all confirmation prompts (requires --mode)
#   -h, --help            show this help
#
# ENVIRONMENT (kept for backwards compatibility):
#   SKIP_RUNNING_PODS_WARNING=true  same as -y for the running-pods prompt
#   MAX_WAIT_SECONDS                how long to wait for the DaemonSet (default 600)

set -u

DAEMONSET_NAME="dynatrace-cleanup-node-fs"
MAX_WAIT_SECONDS="${MAX_WAIT_SECONDS:-600}"
WAIT_BEFORE_DAEMONSET_DESTRUCTION_SECONDS="${WAIT_BEFORE_DAEMONSET_DESTRUCTION_SECONDS:-0}"
# renovate datasource=docker depName=registry.access.redhat.com/ubi9-micro
UBI_MICRO_IMAGE="registry.access.redhat.com/ubi9-micro:9.6-1760515026@sha256:aff810919642215e15c993b9bbc110dbcc446608730ad24499dafd9df7a8f8f4"

NAMESPACE="dynatrace"
MODE=""
CSI_SUBPATH=""
EXTRA_DIRS=()
KUBELET_PATH="${KUBELET_PATH:-/var/lib/kubelet}"
REPORT_DIR=""
ASSUME_YES=false

usage() {
  sed -n '/^# USAGE:/,/^$/{/^$/q;p}' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
  case "$1" in
  -n | --namespace)
    NAMESPACE="$2"
    shift 2
    ;;
  -m | --mode)
    MODE="$2"
    shift 2
    ;;
  --csi-subpath)
    CSI_SUBPATH="$2"
    shift 2
    ;;
  --extra-dir)
    EXTRA_DIRS+=("$2")
    shift 2
    ;;
  --kubelet-path)
    KUBELET_PATH="$2"
    shift 2
    ;;
  --report-dir)
    REPORT_DIR="$2"
    shift 2
    ;;
  -y | --yes)
    ASSUME_YES=true
    shift
    ;;
  -h | --help)
    usage
    exit 0
    ;;
  -*)
    echo "Unknown option: $1" >&2
    usage >&2
    exit 1
    ;;
  *)
    # backwards compatible positional namespace
    NAMESPACE="$1"
    shift
    ;;
  esac
done

if [ "${SKIP_RUNNING_PODS_WARNING:-false}" = true ]; then
  ASSUME_YES=true
fi

case "$MODE" in
classic | csi | all) ;;
"")
  if [ "$ASSUME_YES" = true ] || ! [ -t 0 ]; then
    echo "ERROR: no --mode given and not running interactively. Use --mode classic|csi|all." >&2
    exit 1
  fi
  echo "What do you want to clean up?"
  echo "  1) All - classic host installation AND CSI driver data"
  echo "  2) Classic - host-installed OneAgent only (/opt/dynatrace, /var/lib/dynatrace, /var/log/dynatrace)"
  echo "     Safe alongside a working CSI-based (cloudNativeFullStack/hostMonitoring) install."
  echo "  3) CSI - CSI driver data only (${KUBELET_PATH}/plugins/csi.oneagent.dynatrace.com/data, /var/opt/dynatrace)"
  read -r -p "Enter choice [1-3]: " choice
  case "$choice" in
  1) MODE="all" ;;
  2) MODE="classic" ;;
  3) MODE="csi" ;;
  *)
    echo "Invalid choice. Aborting." >&2
    exit 1
    ;;
  esac
  ;;
*)
  echo "ERROR: invalid mode '$MODE'. Use classic, csi or all." >&2
  exit 1
  ;;
esac

# Build the list of host directories to remove and whether to run the OneAgent
# uninstaller. Paths are host-absolute; the init container prefixes /mnt/root.
RUN_UNINSTALL=false
CLEANUP_DIRS=()

if [ "$MODE" = "classic" ] || [ "$MODE" = "all" ]; then
  RUN_UNINSTALL=true
  CLEANUP_DIRS+=(
    /var/lib/dynatrace
    /opt/dynatrace
    /var/log/dynatrace
  )
fi

if [ "$MODE" = "csi" ] || [ "$MODE" = "all" ]; then
  csi_data_dir="${KUBELET_PATH}/plugins/csi.oneagent.dynatrace.com/data"
  if [ -n "$CSI_SUBPATH" ]; then
    CLEANUP_DIRS+=("${csi_data_dir}/${CSI_SUBPATH}")
  else
    CLEANUP_DIRS+=("$csi_data_dir" /var/opt/dynatrace)
  fi
fi

if [ ${#EXTRA_DIRS[@]} -gt 0 ]; then
  for dir in "${EXTRA_DIRS[@]}"; do
    case "$dir" in
    /*) CLEANUP_DIRS+=("$dir") ;;
    *)
      echo "ERROR: --extra-dir must be an absolute path: $dir" >&2
      exit 1
      ;;
    esac
  done
fi

if [ -z "$REPORT_DIR" ]; then
  REPORT_DIR="./dynatrace-cleanup-report-$(date +%Y%m%d-%H%M%S)"
fi

echo ""
echo "Cleanup plan:"
echo "  Namespace:  $NAMESPACE"
echo "  Mode:       $MODE"
if [ "$RUN_UNINSTALL" = true ]; then
  echo "  Uninstall:  /opt/dynatrace/oneagent/agent/uninstall.sh (if present)"
fi
echo "  Directories to remove on every node:"
printf '    %s\n' "${CLEANUP_DIRS[@]}"
echo "  Report dir: $REPORT_DIR"

if [ "$MODE" != "classic" ]; then
  echo ""
  echo "⚠️  CSI cleanup removes files underneath a potentially running CSI driver."
  echo "    For a real uninstall: delete the DynaKube, restart monitored workloads, remove the"
  echo "    operator (incl. CSI driver) first. Running this against a live install is intended"
  echo "    for support/troubleshooting only and can break injected pods."
fi

if [ "$ASSUME_YES" != true ]; then
  echo ""
  read -r -p "Proceed? (yes/no): " response
  case "$response" in
  [yY][eE][sS] | [yY]) ;;
  *)
    echo "Cleanup cancelled."
    exit 0
    ;;
  esac
fi

echo ""
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

  if [ "$ASSUME_YES" = true ]; then
    echo "Skipping running pods warning. Continuing with cleanup..."
  else
    read -r -p "Do you want to continue anyway? (yes/no): " response
    case "$response" in
    [yY][eE][sS] | [yY])
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
      # hostPID is needed so the classic-mode fallback can find and kill
      # lingering host OneAgent processes
      hostPID: true
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
            report=""

            echo 'Starting node filesystem cleanup (mode: ${MODE})...';

            if [ "${RUN_UNINSTALL}" = "true" ]; then
                if [ -f /mnt/root/opt/dynatrace/oneagent/agent/uninstall.sh ]; then
                    echo 'Executing OneAgent uninstall script...';
                    chroot /mnt/root /opt/dynatrace/oneagent/agent/uninstall.sh
                    rc=\$?
                    if [ \$rc -ne 0 ]; then
                        echo "ERROR: OneAgent uninstall script failed with exit code \$rc";
                        report="\${report}UNINSTALL failed (exit \$rc)\n"
                        exit_code=1
                    else
                        echo 'OneAgent uninstall script completed successfully.';
                        report="\${report}UNINSTALL ok\n"
                    fi
                else
                    echo 'OneAgent uninstall script not found, skipping...';
                    report="\${report}UNINSTALL not-found\n"
                fi

                # Remove system-level residue the uninstaller would normally handle.
                # Idempotent, so it also runs when uninstall.sh succeeded.
                # NOTE: no grep in ubi-micro, so lines are matched with shell patterns.
                preload=/mnt/root/etc/ld.so.preload
                preload_scrubbed=false
                if [ -f "\$preload" ]; then
                    : > "\$preload.tmp"
                    while IFS= read -r line || [ -n "\$line" ]; do
                        case "\$line" in
                        *dynatrace* | *oneagent*) preload_scrubbed=true ;;
                        *) printf '%s\n' "\$line" >> "\$preload.tmp" ;;
                        esac
                    done < "\$preload"
                    if [ "\$preload_scrubbed" = true ]; then
                        if [ -s "\$preload.tmp" ]; then
                            cat "\$preload.tmp" > "\$preload"
                            rm -f "\$preload.tmp"
                        else
                            rm -f "\$preload" "\$preload.tmp"
                        fi
                        echo 'Removed oneagent entries from /etc/ld.so.preload'
                    else
                        rm -f "\$preload.tmp"
                    fi
                fi
                if [ "\$preload_scrubbed" = true ]; then
                    report="\${report}PRELOAD scrubbed\n"
                else
                    report="\${report}PRELOAD clean\n"
                fi

                chroot /mnt/root systemctl stop oneagent.service >/dev/null 2>&1
                chroot /mnt/root systemctl disable oneagent.service >/dev/null 2>&1
                units_removed=0
                for unit in /mnt/root/etc/systemd/system/oneagent*.service \
                            /mnt/root/etc/systemd/system/*/oneagent*.service \
                            /mnt/root/usr/lib/systemd/system/oneagent*.service; do
                    if [ -e "\$unit" ]; then
                        rm -f "\$unit"
                        units_removed=\$((units_removed + 1))
                        echo "Removed systemd unit: \${unit#/mnt/root}"
                    fi
                done
                if [ \$units_removed -gt 0 ]; then
                    chroot /mnt/root systemctl daemon-reload >/dev/null 2>&1
                    report="\${report}UNITS removed \$units_removed\n"
                else
                    report="\${report}UNITS none\n"
                fi

                # Kill host processes still running from dynatrace paths. Requires hostPID.
                # CSI-based agents report the same exe paths from inside their containers,
                # so only processes sharing the host's root filesystem are killed.
                host_root=\$(stat -c %d:%i /proc/1/root/ 2>/dev/null)
                kill_dynatrace_procs() {
                    count=0
                    for pidpath in /proc/[0-9]*; do
                        pid=\${pidpath#/proc/}
                        exe=\$(readlink "\$pidpath/exe" 2>/dev/null) || continue
                        case "\$exe" in
                        /opt/dynatrace/* | /var/lib/dynatrace/*) ;;
                        *) continue ;;
                        esac
                        [ "\$(stat -c %d:%i "\$pidpath/root/" 2>/dev/null)" = "\$host_root" ] || continue
                        kill "-\$1" "\$pid" 2>/dev/null && count=\$((count + 1))
                    done
                    echo \$count
                }
                killed=\$(kill_dynatrace_procs TERM)
                if [ "\$killed" -gt 0 ]; then
                    echo "Sent SIGTERM to \$killed dynatrace host process(es), waiting before SIGKILL..."
                    sleep 5
                    kill_dynatrace_procs KILL >/dev/null
                    report="\${report}PROCS killed \$killed\n"
                else
                    report="\${report}PROCS none\n"
                fi
            fi

            echo 'Removing directories if they exist...';
            directories="${CLEANUP_DIRS[*]}"

            for dir in \$directories; do
                hostdir="/mnt/root\$dir"
                if [ -d "\$hostdir" ]; then
                    size=\$(du -sh "\$hostdir" 2>/dev/null | cut -f1)
                    echo "Inventory of \$dir (\${size:-?}) before removal:"
                    ls -laR "\$hostdir" 2>/dev/null | head -200
                    if rm -rf "\$hostdir" 2>&1; then
                        if [ -d "\$hostdir" ]; then
                            echo "ERROR: \$dir still present after removal"
                            report="\${report}LEFTOVER \$dir\n"
                            exit_code=1
                        else
                            echo "Removed directory: \$dir"
                            report="\${report}REMOVED \$dir (\${size:-?})\n"
                        fi
                    else
                        echo "ERROR: Failed to remove directory: \$dir"
                        report="\${report}FAILED \$dir\n"
                        exit_code=1
                    fi
                else
                    echo "Directory not found (skipping): \$dir"
                    report="\${report}MISSING \$dir\n"
                fi
            done

            if [ \$exit_code -eq 0 ]; then
                echo 'SUCCESS: Node filesystem cleanup completed successfully.';
                printf 'SUCCESS\n%b' "\$report" > /dev/termination-log
            else
                echo 'FAILURE: Node filesystem cleanup completed with errors.';
                printf 'FAILURE\n%b' "\$report" > /dev/termination-log
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
while [ $elapsed -lt "$MAX_WAIT_SECONDS" ]; do
  desired=$(kubectl get daemonset "$DAEMONSET_NAME" -n "$NAMESPACE" -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
  ready=$(kubectl get daemonset "$DAEMONSET_NAME" -n "$NAMESPACE" -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
  current=$(kubectl get daemonset "$DAEMONSET_NAME" -n "$NAMESPACE" -o jsonpath='{.status.currentNumberScheduled}' 2>/dev/null || echo "0")

  printf "\rDaemonSet status: Ready: %s/%s (Current scheduled: %s) - %ss elapsed" "$ready" "$desired" "$current" "$elapsed"

  if [ "$ready" -eq "$desired" ] && [ "$desired" -gt "0" ]; then
    echo ""
    echo "✅ All $desired DaemonSet pods are ready."
    break
  fi

  sleep 5
  elapsed=$((elapsed + 5))
done

if [ $elapsed -ge "$MAX_WAIT_SECONDS" ]; then
  echo ""
  echo "⚠️  Timeout waiting for all pods to be running"
fi

# Verify node coverage: the DaemonSet silently skips nodes whose taints it does not
# tolerate, so "ready == desired" alone is NOT proof that every node was cleaned.
echo ""
echo "Verifying node coverage..."
all_nodes=$(kubectl get nodes -l kubernetes.io/os=linux --no-headers -o custom-columns=":metadata.name" | sort)
covered_nodes=$(kubectl get pods -n "$NAMESPACE" -l app="$DAEMONSET_NAME" --no-headers -o custom-columns=":spec.nodeName" | sort)
uncovered_nodes=$(comm -23 <(echo "$all_nodes") <(echo "$covered_nodes"))
total_nodes=$(echo "$all_nodes" | grep -c . || true)
covered_count=$(echo "$covered_nodes" | grep -c . || true)

if [ -n "$uncovered_nodes" ]; then
  echo "⚠️  WARNING: $covered_count of $total_nodes Linux nodes got a cleanup pod."
  echo "    The following nodes were NOT cleaned (check their taints/architecture):"
  for node in $uncovered_nodes; do
    taints=$(kubectl get node "$node" -o jsonpath='{range .spec.taints[*]}{.key}={.value}:{.effect} {end}' 2>/dev/null)
    echo "      - $node ${taints:+(taints: $taints)}"
  done
else
  echo "✅ All $total_nodes Linux nodes are covered by a cleanup pod."
fi

# Get detailed status
echo ""
echo "Pod status summary:"
kubectl get pods -n "$NAMESPACE" -l app="$DAEMONSET_NAME" -o wide

echo ""
echo "Collecting per-node logs and pod manifests into $REPORT_DIR ..."
mkdir -p "$REPORT_DIR"

echo ""
echo "Init container completion status:"
successful_inits=0
failed_inits=0
total_pods=0

while read -r pod node; do
  [ -n "$pod" ] || continue
  total_pods=$((total_pods + 1))
  node="${node:-$pod}"

  kubectl logs "$pod" -n "$NAMESPACE" -c cleanup-init >"$REPORT_DIR/$node.log" 2>&1
  kubectl get pod "$pod" -n "$NAMESPACE" -o yaml >"$REPORT_DIR/$node.yaml" 2>&1

  # Check if init container completed (it always exits 0 to prevent restarts)
  init_status=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.initContainerStatuses[0].state}' 2>/dev/null)

  if echo "$init_status" | grep -q "terminated"; then
    # The termination message starts with SUCCESS or FAILURE, followed by a per-path report
    termination_message=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.initContainerStatuses[0].state.terminated.message}' 2>/dev/null)
    result=$(echo "$termination_message" | head -n 1)

    if [ "$result" = "SUCCESS" ]; then
      successful_inits=$((successful_inits + 1))
      echo "  ✅ $node"
    elif [ "$result" = "FAILURE" ]; then
      failed_inits=$((failed_inits + 1))
      echo "  ❌ $node - cleanup failed (full log: $REPORT_DIR/$node.log)"
    else
      echo "  ⚠️  $node - unknown status (termination message: '$termination_message')"
    fi
    echo "$termination_message" | tail -n +2 | sed 's/^/       /'
  else
    echo "  ⏳ $node - init container still running or pending"
  fi
done < <(kubectl get pods -n "$NAMESPACE" -l app="$DAEMONSET_NAME" --no-headers -o custom-columns=":metadata.name,:spec.nodeName")

echo ""
echo "📊 Cleanup Results:"
echo "  ✅ Successful cleanups: $successful_inits/$total_pods"
if [ $failed_inits -gt 0 ]; then
  echo "  ❌ Failed cleanups: $failed_inits/$total_pods"
fi
if [ -n "$uncovered_nodes" ]; then
  echo "  ⚠️  Nodes without a cleanup pod: $((total_nodes - covered_count))"
fi
echo "  📁 Logs and pod manifests: $REPORT_DIR"

sleep "$WAIT_BEFORE_DAEMONSET_DESTRUCTION_SECONDS"

# Only delete the DaemonSet if all cleanups were successful
daemonset_deleted=false
if [ $failed_inits -eq 0 ] && [ $successful_inits -eq $total_pods ] && [ $total_pods -gt 0 ]; then
  echo ""
  echo "✅ All cleanups successful. Deleting cleanup DaemonSet..."
  kubectl delete daemonset "$DAEMONSET_NAME" -n "$NAMESPACE"
  daemonset_deleted=true
else
  echo ""
  echo "⚠️  Some cleanups failed or are incomplete. Keeping DaemonSet for investigation."
  echo "    To view logs: kubectl logs -n $NAMESPACE -l app=$DAEMONSET_NAME -c cleanup-init"
  echo "    To delete manually: kubectl delete daemonset $DAEMONSET_NAME -n $NAMESPACE"
fi

echo ""
echo "Restarting CSI driver pods in case they are deployed..."
kubectl -n "$NAMESPACE" delete pod -l app.kubernetes.io/component=csi-driver,app.kubernetes.io/name=dynatrace-operator

if [ "$namespace_created" = true ]; then
  if [ "$daemonset_deleted" = true ]; then
    echo ""
    echo "Deleting namespace $NAMESPACE (created by this script)..."
    kubectl delete namespace "$NAMESPACE" --ignore-not-found
  else
    echo ""
    echo "⚠️  Keeping namespace $NAMESPACE so the DaemonSet stays available for investigation."
    echo "    To delete it later: kubectl delete namespace $NAMESPACE"
  fi
fi

echo ""
echo "Cleanup process completed."
