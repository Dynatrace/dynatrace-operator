#!/usr/bin/env python3
import argparse
import yaml


def _travers_yaml(input_yaml, path):
    property_path = path.split(".")
    property = input_yaml
    for i in range(0, len(property_path) - 1):
        property = property[property_path[i]]
    return property


def _get_final_property_name(path):
    property_path = path.split(".")
    return property_path[-1]


if __name__ == "__main__":
    argument_parser = argparse.ArgumentParser(description="Sets a property in a YAML file to the specified version")
    argument_parser.add_argument("--file", type=str, required=True,
                                 help="The file in which the property is changed")
    argument_parser.add_argument("--value", type=str, default=None,
                                 help="The value to which the property is set")
    argument_parser.add_argument("--value-from-file", type=str, default=None,
                                 help="Read the value from this file instead of --value")
    argument_parser.add_argument("--property", type=str, default="version",
                                 help="The name of the property to be changed. "
                                      "Can be specified as a path seperated by dots."
                                      "E.g. rootProperty.versionProperty")
    argument_parser.add_argument("--output-file", type=str, required=False, default=None,
                                 help="The file to which the output is written to. Defaults to the input file")

    args = argument_parser.parse_args()

    if args.output_file is None:
        args.output_file = args.file

    file = args.file
    output_file = args.output_file

    if args.value_from_file is not None:
        with open(args.value_from_file, "r") as f:
            value = f.read()
    else:
        value = args.value

    property_name = args.property

    with open(file, "r") as input_stream:
        input_yaml = yaml.safe_load(input_stream)

    version_property = _travers_yaml(input_yaml, property_name)

    if value is not None:
        version_property[_get_final_property_name(property_name)] = value
    else:
        version_property.pop(_get_final_property_name(property_name), None)

    with open(output_file, "w") as output_stream:
        yaml.safe_dump(input_yaml, output_stream, sort_keys=False)
