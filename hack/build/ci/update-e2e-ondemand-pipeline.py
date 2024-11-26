import sys

from ruamel.yaml import YAML

# version file contains a list of strings
version_file = "release-branches.txt"
ondemand_file = ".github/workflows/e2e-tests-ondemand.yaml"

version = ""
# read versions to list
with open(version_file, "r") as f:
    version = [v.strip().replace("origin/", "") for v in f.readlines()][-1]

yaml = YAML()
yaml.width = 4096

# read ondemand_file file to dict and update
with open(ondemand_file, "r") as f:
    data = yaml.load(f)
    data["env"]["branch"] = version

print(data)
# write ondemand_file renovate file
with open(ondemand_file, "wb") as output:
    yaml.indent(mapping=2, sequence=4, offset=2)
    yaml.dump(data, output)
