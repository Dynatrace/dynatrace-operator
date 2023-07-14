import sys
from urllib.parse import urlparse
import yaml
import argparse
from urllib.request import urlopen

apiTerms = {
    "create": "Create",
    "get": "Get",
    "list": "List",
    "watch": "Watch",
    "update": "Update",
    "delete": "Delete",
    "patch": "Patch",
    "use": "Use"
}

resourceTerms = {
    "nodes": "Nodes",
    "pods": "Pods",
    "namespaces": "Namespaces",
    "replicationcontrollers": "ReplicationControllers",
    "events": "Events",
    "resourcequotas": "ResourceQuotas",
    "pods/proxy": "Pods/Proxy",
    "nodes/proxy": "Nodes/Proxy",
    "nodes/metrics": "Nodes/Metrics",
    "services": "Services",
    "jobs": "Jobs",
    "cronjobs": "CronJobs",
    "deployments": "Deployments",
    "replicasets": "ReplicaSets",
    "statefulsets": "StatefulSets",
    "daemonsets": "DaemonSets",
    "deploymentconfigs": "DeploymentConfigs",
    "clusterversions": "ClusterVersions",
    "secrets": "Secrets",
    "mutatingwebhookconfigurations": "MutatingWebhookConfigurations",
    "validatingwebhookconfigurations": "ValidatingWebhookConfigurations",
    "customresourcedefinitions": "CustomResourceDefinitions",
    "csinodes": "CsiNodes",
    "dynakubes": "Dynakubes",
    "dynakubes/finalizers": "Dynakubes/Finalizers",
    "dynakubes/status": "Dynakubes/Status",
    "deployments/finalizers": "Deployments/Finalizers",
    "configmaps": "ConfigMaps",
    "pods/log": "Pods/Log",
    "servicemonitors": "ServiceMonitors",
    "serviceentries": "ServiceEntries",
    "virtualservices": "VirtualServices",
    "leases": "Leases",
    "endpoints": "EndPoints",
    "securitycontextconstraints": "SecurityContextConstraints"
}

sectionTitles = {
    "dynatrace-operator": "Dynatrace Operator",
    "dynatrace-kubernetes-monitoring": "Dynatrace Kubernetes Monitoring (ActiveGate)",
    "dynatrace-webhook": "Dynatrace webhook server",
    "dynatrace-oneagent-csi-driver": "Dynatrace CSI driver",
    "dynatrace-activegate": "Dynatrace Kubernetes Monitoring (ActiveGate)",
    "dynatrace-dynakube-oneagent": "Dynatrace OneAgent"
}

def get_apis(rule):
    apis = rule.get('verbs')

    api_string = ""
    for api in apis:
        if (len(api_string) > 0):
            api_string += "/"
        api_string += apiTerms[api]

    return api_string

def multiline_codestyle_block(stringList):
    result_string = ""
    for entry in stringList:
        if (len(result_string) > 0):
            result_string += "<br />"
        if len(entry) > 0:
            result_string += f"`{entry}`"
        else:
            result_string += f"`-`"
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
                api_gropus = get_api_groups(rule)
                print(f"|`{resourceTerms[resource]}` |{api_gropus} |{apis} |{resource_names} |")

def convert_cluster_roles_to_markdown(role):
    print(f"\n## {sectionTitles[role['metadata']['name']]} (cluster-wide)\n")
    create_role_table(role)

def convert_roles_to_markdown(role):
    print(f"\n## {sectionTitles[role['metadata']['name']]} (namespace {role['metadata']['namespace']})\n")
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

    docs = yaml.safe_load_all(file)

    # for manifest in docs:
    #     print(f"{manifest['metadata']['name']} {manifest['kind']}")
    manifests = []

    for manifest in docs:
        manifests.append(manifest)

    for manifest in manifests:
#        print(f"\n{manifest['metadata']['name']} {manifest['kind']}")
        if manifest['kind'] == 'ClusterRole':
            convert_cluster_roles_to_markdown(manifest)

    for manifest in manifests:
        #        print(f"\n ** {manifest['metadata']['name']} {manifest['kind']}")
        if manifest['kind'] == 'Role':
            convert_roles_to_markdown(manifest)

    file.close()
if __name__ == "__main__":
    sys.exit(main())
