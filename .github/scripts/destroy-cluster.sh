#!/usr/bin/env bash

set -x

echo "Destroying environment '$FLC_ENVIRONMENT' in namespace '$FLC_NAMESPACE'"

kubectl get flcenvironments --namespace "$FLC_NAMESPACE"

echo "Patching environment '$FLC_ENVIRONMENT' to 'not-deployed'"
kubectl patch --namespace "$FLC_NAMESPACE" --type merge --patch '{"spec": {"desiredState": "environment-not-deployed"}}' flcenvironment "$FLC_ENVIRONMENT"

echo "Waiting up to 20m for successful destruction of environment '$FLC_ENVIRONMENT'"

# wait until timeout or until the environment is in the desired state
NEXT_WAIT_TIME=0
while [[ $NEXT_WAIT_TIME -ne 20 ]]; do
  # Check if the environment is in the desired state
  current_state=$(kubectl get flcenvironment "$FLC_ENVIRONMENT" --namespace "$FLC_NAMESPACE" -ojsonpath='{.status.currentState}')

  if [[ "$current_state" == "environment-not-deployed" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' has been destroyed successfully."
    exit 0
  elif [[ "$current_state" == "environment-destruction-failed" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' destruction is failed. Please check the logs for more details."
    exit 1
  elif [[ -z "$current_state" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' does not exist or is not ready. Exiting."
    exit 1
  else
    echo "Current state of environment '$FLC_ENVIRONMENT': '$current_state'. Waiting for desired state 'environment-not-deployed'..."
    NEXT_WAIT_TIME=$((NEXT_WAIT_TIME+1))
    sleep 60
  fi
done

echo "Timeout reached while waiting for environment '$FLC_ENVIRONMENT' to be destroyed."
exit 1
