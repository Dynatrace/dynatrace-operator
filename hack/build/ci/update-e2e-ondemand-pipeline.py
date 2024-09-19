import sys

from ruamel.yaml import YAML

# version file contains a list of strings
version_file = sys.argv[1]
ondemand_file = sys.argv[2]

version = ""
# read versions to list
with open(version_file, "r") as f:
    version = f.readline().strip().replace("origin/", "")

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
