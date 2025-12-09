#!/usr/bin/env python3
import argparse
import sys
import datetime
import yaml


def read_yaml(path):
    with open(path, "r") as yaml_file:
        try:
            data = yaml.safe_load(yaml_file)
        except yaml.YAMLError as e:
            print(f"Could not load file: {e}")
            return None
    return data


def write_yaml(data, path):
    with open(path, "w") as file:
        yaml.dump(data, file, default_flow_style=False, sort_keys=False)


if __name__ == "__main__":
    argument_parser = argparse.ArgumentParser(description="Finalize CSV files by automatically setting the createdAt annotation and olm.skipRange")
    argument_parser.add_argument("--platform", type=str, required=True, choices=["openshift", "kubernetes"],
                                 help="Sets the platform for which the CSV files are finalized")
    argument_parser.add_argument("--version", type=str, required=True,
                                 help="Sets the version for which the CSV files are finalized")

    args = argument_parser.parse_args()

    platform = args.platform
    version = args.version
    csv_filepath = \
        f"config/olm/{platform}/{version}/manifests/" \
        f"dynatrace-operator.clusterserviceversion.yaml"
    kustomize_filepath = f"config/olm/{platform}/kustomization.yaml"

    csv = read_yaml(csv_filepath)
    kustomize = read_yaml(kustomize_filepath)

    if csv is None or kustomize is None:
        sys.exit(1)

    kustomize.pop("images", None)
    csv["metadata"]["annotations"]["createdAt"] = datetime.datetime.now().isoformat()

    for deployment_index, deployment in enumerate(csv["spec"]["install"]["spec"]["deployments"]):
        for container_index, container in enumerate(deployment["spec"]["template"]["spec"]["containers"]):
            deployment["spec"]["template"]["spec"]["containers"][container_index] = container
        csv["spec"]["install"]["spec"]["deployments"][deployment_index] = deployment

    csv["metadata"]["annotations"]["olm.skipRange"] = f"<{version}"

    write_yaml(csv, csv_filepath)
    write_yaml(kustomize, kustomize_filepath)