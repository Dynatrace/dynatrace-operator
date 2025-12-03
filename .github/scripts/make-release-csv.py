#!/usr/bin/env python3

import argparse
import yaml

def prepare_for_RHCC(csv):
    csv["metadata"]["annotations"]["marketplace.openshift.io/remote-workflow"] = \
        "https://marketplace.redhat.com/en-us/operators/dynatrace-operator-rhmp/pricing?utm_source=openshift_console"
    csv["metadata"]["annotations"]["marketplace.openshift.io/support-workflow"] = \
        "https://marketplace.redhat.com/en-us/operators/dynatrace-operator-rhmp/support?utm_source=openshift_console"

    return csv

def configure_deployment(deployment, image, marketplace):
    containers = deployment["spec"]["template"]["spec"]["containers"]
    for container_index in range(len(containers)):
        containers[container_index]["image"] = image
    
    # Setting changed array since I am unsure if python assigns arrays by reference or value
    deployment["spec"]["template"]["containers"] = containers

    if deployment["name"] == "dynatrace-operator":
        formattedMarketplace = "operatorhub" + "-" + marketplace
        deployment["spec"]["template"]["metadata"]["labels"]["dynatrace.com/install-source"] = formattedMarketplace

    return deployment

if __name__ == "__main__":
    argument_parser = argparse.ArgumentParser()
    argument_parser.add_argument("path", type=str)
    argument_parser.add_argument("--image", type=str, required=True)
    argument_parser.add_argument("--isRHCC", type=bool, default=False)
    argument_parser.add_argument("--marketplace", type=str, required=True)

    arguments = argument_parser.parse_args()
    csv_path = arguments.path
    image = arguments.image
    isRHCC = arguments.isRHCC
    marketplace = arguments.marketplace

    with open(csv_path, 'r') as csv_file:
        csv = yaml.safe_load(csv_file)

        if isRHCC:
            csv = prepare_for_RHCC(csv)

        csv["metadata"]["annotations"]["containerImage"] = image
        
        deployments = csv["spec"]["install"]["spec"]["deployments"]
        for deployment_index in range(len(deployments)):
            deployments[deployment_index] = configure_deployment(deployments[deployment_index], image, marketplace)

        # Setting changed array since I am unsure if python assigns arrays by reference or value
        csv["spec"]["install"]["spec"]["deployments"] = deployments

        csv["metadata"]["annotations"]["operators.openshift.io/valid-subscription"] = \
        '[\"Dynatrace Platform Subscription (DPS)\",\"Dynatrace Classic License\"]'
        
        csv["metadata"]["annotations"]["features.operators.openshift.io/disconnected"] = "true"
        csv["metadata"]["annotations"]["features.operators.openshift.io/proxy-aware"] = "true"
        csv["metadata"]["annotations"]["features.operators.openshift.io/fips-compliant"] = "false"
        csv["metadata"]["annotations"]["features.operators.openshift.io/tls-profiles"] = "false"
        csv["metadata"]["annotations"]["features.operators.openshift.io/token-auth-aws"] = "false"
        csv["metadata"]["annotations"]["features.operators.openshift.io/token-auth-azure"] = "false"
        csv["metadata"]["annotations"]["features.operators.openshift.io/token-auth-gcp"] = "false"

        
        csv["spec"]["relatedImages"] = [
            {
                "name": "dynatrace-operator",
                "image": image
            }
        ]
            
    with open(csv_path, "w") as csv_file:
        yaml.safe_dump(csv, csv_file, sort_keys=False)
        
