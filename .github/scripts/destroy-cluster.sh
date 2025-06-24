#!/usr/bin/env bash

set -x

DEFAULT_TIMEOUT="20m"
DESIRED_STATE="environment-not-deployed"

echo "Destroying environment '$FLC_ENVIRONMENT' in namespace '$FLC_NAMESPACE'"

kubectl get flcenvironments --namespace "$FLC_NAMESPACE"

echo "Patching environment '$FLC_ENVIRONMENT' to 'not-deployed'"
kubectl patch --namespace "$FLC_NAMESPACE" --type merge --patch '{"spec": {"desiredState": "environment-not-deployed"}}' flcenvironment "$FLC_ENVIRONMENT"

# wait until timeout or until the environment is in the desired state
NEXT_WAIT_TIME=0
while [[ $NEXT_WAIT_TIME -ne 20 ]]; do
  # Check if the environment is in the desired state
  current_state=$(kubectl --namespace "$FLC_NAMESPACE" --for jsonpath='{.status.currentState}' flcenvironment "$FLC_ENVIRONMENT")

  if [[ "$current_state" == "$DESIRED_STATE" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' is in desired state '$DESIRED_STATE'."
    break
  elif [[ "$current_state" == "environment-destruction-failed" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' destruction is failed. Please check the logs for more details."
    exit 1
  elif [[ -z "$current_state" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' does not exist or is not ready. Exiting."
    exit 1
  else
    echo "Current state of environment '$FLC_ENVIRONMENT': '$current_state'. Waiting for desired state '$DESIRED_STATE'..."
    let NEXT_WAIT_TIME += 1
    sleep 60
  fi
done
