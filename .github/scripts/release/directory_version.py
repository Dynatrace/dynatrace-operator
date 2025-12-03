#!/usr/bin/env python3
import argparse

from version.collection import get_latest_version_from_collection

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
