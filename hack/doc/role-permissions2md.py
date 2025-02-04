import sys
from urllib.parse import urlparse
import yaml
import argparse
from urllib.request import urlopen

def get_apis(rule):
    apis = rule.get('verbs')

    api_string = ""
    for api in apis:
        if (len(api_string) > 0):
            api_string += "/"
        api_string += api.title()

    return api_string

def multiline_codestyle_block(entries):
    result_string = ""
    for entry in entries:
        if (len(result_string) > 0):
            result_string += "<br />"
        if len(entry) > 0:
            result_string += f"`{entry}`"
        else:
            result_string += f"`\"\"`"
    return result_string

def get_resource_names(rule):
    resource_names = rule.get('resourceNames')
    if resource_names == None:
        return ""
    return multiline_codestyle_block(resource_names)

def get_api_groups(rule):
    api_groups = rule.get("apiGroups")
    if api_groups == None:
        return ""
    return multiline_codestyle_block(api_groups)

def create_role_table(role):
    print('|Resources accessed |API group |APIs used |Resource names |')
    print('|------------------ |--------- |--------- |-------------- |')

    for rule in role['rules']:
        resources = rule.get('resources')
        if resources != None:
            for resource in resources:
                apis = get_apis(rule)
                resource_names = get_resource_names(rule)
                api_groups = get_api_groups(rule)
                print(f"|`{resource}` |{api_groups} |{apis} |{resource_names} |")

def beautify_section(role):
    section = role['metadata']['name']
    section = section.replace("-dynakube", "").replace("-", " ").title().replace("Csi", "CSI")

    return section

def convert_role_to_markdown(role):
    scope = "Namespaced"
    if role['kind'] == 'ClusterRole':
        scope = "Cluster-wide"

    print(f"\n## {beautify_section(role)} ({scope})\n")
    create_role_table(role)


def main():
    parser = argparse.ArgumentParser(description="Convert ClusterRoles and Roles to MD permission table",
                                    formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument("src", help="Source K8S manifest")
    args = parser.parse_args()

    parsed_url = urlparse(args.src)
    if bool(parsed_url.scheme):
        file = urlopen(args.src)
    else:
        file = open(args.src, "r")

    try:
        docs = yaml.safe_load_all(file)

        core_manifests = []
        other_manifests = []
        for manifest in docs:
            if not manifest['kind'] in ("Role", "ClusterRole"):
                continue
            
            if any(f in manifest["metadata"]["name"] for f in ("operator", "webhook", "csi-driver")):
                core_manifests.append(manifest)
            else:
                other_manifests.append(manifest)

        core_manifests.sort(key=lambda m: m["metadata"]["name"])
        other_manifests.sort(key=lambda m: m["metadata"]["name"])
        
        for manifest in core_manifests:
            convert_role_to_markdown(manifest)
        for manifest in other_manifests:
            convert_role_to_markdown(manifest)

    finally:
        file.close()

if __name__ == "__main__":
    sys.exit(main())
