#!/usr/bin/env python3
import argparse


def _compare_normalized_version_arrays(a, b):
    a_is_parseable = True
    b_is_parseable = True
    for i in range(0, 3):
        version_num_a = 0
        version_num_b = 0

        try:
            version_num_a = int(a[i])
        except ValueError:
            a_is_parseable = False
        try:
            version_num_b = int(b[i])
        except ValueError:
            b_is_parseable = False

        if not a_is_parseable and not b_is_parseable:
            return 0
        elif not a_is_parseable:
            return -1
        elif not b_is_parseable:
            return 1

        if version_num_a > version_num_b:
            return 1
        elif version_num_b > version_num_a:
            return -1
    return 0


def normalize_version(version):
    version_array = _split_semantic_version(version)
    version_array = _normalize_version_array(version_array)
    return ".".join(version_array)


def _split_semantic_version(version):
    return version.split(".")


def _normalize_version_array(version_array):
    while len(version_array) < 3:
        version_array.append("0")
    return version_array[0:3]


def get_latest_version_from_collection(version_collection):
    if len(version_collection) <= 0:
        return ""

    if len(version_collection) == 1:
        return version_collection[0]

    latest_version = ""
    for version in version_collection:
        latest = _normalize_version_array(_split_semantic_version(latest_version))
        current = _normalize_version_array(_split_semantic_version(version))
        comparison = _compare_normalized_version_arrays(latest, current)

        if comparison < 0:
            latest_version = version

    return normalize_version(latest_version)


if __name__ == "__main__":
    argument_parser = argparse.ArgumentParser(
        description="Returns the latest version from a list of strings in the semver format")
    argument_parser.add_argument("versions",
                                 help="Newline separated list of versions. E.g. the output of 'ls -d */'. "
                                      "Trailing slashes will be removed for your convenience",
                                 type=str)
    arguments = argument_parser.parse_args()

    versions_string = arguments.versions
    version_dirs = versions_string.split("\n")
    versions = []

    for version_dir in version_dirs:
        if version_dir.endswith("/"):
            version_dir = version_dir[:-1]
        versions.append(version_dir.split("/")[-1])

    print(get_latest_version_from_collection(versions))
