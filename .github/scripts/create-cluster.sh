#!/usr/bin/env bash

set -x

DEFAULT_TIMEOUT="80m" # K8s takes <10min, Openshift >40min
DESIRED_STATE="environment-deployed"

kubectl version

echo "Creating environment '$FLC_ENVIRONMENT' in namespace '$FLC_NAMESPACE'"

kubectl get flcenvironments --namespace "$FLC_NAMESPACE"

echo "Patching environment '$FLC_ENVIRONMENT' to 'deployed'"
kubectl patch --namespace "$FLC_NAMESPACE" --type merge --patch '{"spec": {"desiredState": "environment-deployed"}}' flcenvironment "$FLC_ENVIRONMENT"

echo "Waiting up to '$DEFAULT_TIMEOUT' for successful deployment of environment '$FLC_ENVIRONMENT'"

# wait until timeout or until the environment is in the desired state
NEXT_WAIT_TIME=0
while [[ $NEXT_WAIT_TIME -ne 80 ]]; do
  # Check if the environment is in the desired state
  current_state=$(kubectl get flcenvironment "$FLC_ENVIRONMENT" --namespace "$FLC_NAMESPACE" -ojsonpath='{.status.currentState}')

  if [[ "$current_state" == "environment-deployed" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' has been deployed successfully."
    break
  elif [[ "$current_state" == "environment-deployment-failed" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' deployment failed. Please check the logs for more details."
    exit 1
  elif [[ -z "$current_state" ]]; then
    echo "Environment '$FLC_ENVIRONMENT' does not exist or is not ready. Exiting."
    exit 1
  else
    echo "Current state of environment '$FLC_ENVIRONMENT': '$current_state'. Waiting for desired state 'environment-deployed'..."
    NEXT_WAIT_TIME=$((NEXT_WAIT_TIME+1))
    sleep 60
  fi
done

echo "Timeout reached while waiting for environment '$FLC_ENVIRONMENT' to be deployed."
exit 1
