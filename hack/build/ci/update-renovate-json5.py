import json5

# version file contains a list of strings
versionFile = "release-branches.txt"
renovateFile = ".github/renovate.json5"

# read versions to list
versions = ["$default"]
with open(versionFile, "r") as f:
    versions += f.readlines()
    versions = [x.strip() for x in versions]
    versions = [x.replace("origin/", "") for x in versions]

# read renovate file to dict and update
with open(renovateFile, "r") as f:
    data = json5.load(f)

    data["baseBranches"] = versions
    data["packageRules"][0]["matchBaseBranches"] = versions

# write updated renovate file
with open(renovateFile, "w") as output:
    json5.dump(data, output, indent=2)
    # editorconfig is set up to add a newline to files
    output.write('\n')
