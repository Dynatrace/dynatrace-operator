import argparse
from collections import defaultdict

import yaml


DEPRICATED_FIELDS = ["routing", "kubernetesMonitoring"]


def table_header():
    return "|Parameter|Description|Default value|Data type|"


def main():
    parser = argparse.ArgumentParser(
        description="Convert CR parameters to documentation table",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "crd_path", default="/config/crd/bases/dynatrace.com_dynakubes.yaml", help="path to crd to get OpenAPI spec"

    )
    args = parser.parse_args()

    with open(args.crd_path, "r") as file:
        crd = yaml.safe_load(file)

    max_version_index = max(len(crd["spec"]["versions"]) - 1, 0)
    spec = crd["spec"]["versions"][max_version_index]["schema"]["openAPIV3Schema"]

    print("## {name} schema".format(name=crd["spec"]["names"]["kind"]))

    props = spec["properties"]["spec"]["properties"]
    d = defaultdict(list)
    for lvl, p, obj in traverse(props):
        d[lvl].append((p, obj))

    for k in sorted(d, key=len):
        res = [f"\n### {k}\n", table_header(), "|:-|:-|:-|:-|"]
        for name, obj in d[k]:
            raw_desc = obj.get("description", "")

            if "anyOf" in obj:
                any_type = []
                for subtype in obj["anyOf"]:
                    any_type.append(subtype["type"])
                type_ = " or ".join(any_type)
            else:
                type_ = obj["type"]

            template = "|{field}|{description}|{default}|{type}|".format(
                field=f"`{name}`",
                type=type_,
                description=clean_description(raw_desc),
                default=obj.get("default", "-"),
            )
            res.append(template)
        print("\n".join(res))


def traverse(props, level=".spec"):
    for prop in props:
        if prop in DEPRICATED_FIELDS:
            continue

        pp = props[prop]
        if "properties" in pp and (
            "x-kubernetes-map-type" not in pp
            # some objects we don't want to unfold
            and "requests" not in pp["properties"]
            and "value" not in pp["properties"]
        ):
            for lvl, p, _p in traverse(
                props[prop]["properties"], level=level + "." + prop
            ):
                yield lvl, p, _p
        else:
            yield level, prop, props[prop]


def clean_description(desc):
    d = '<br/>'.join(desc.splitlines())
    d = d.replace('\t', '  ')

    if 'http' in d:
        return d.replace('(', '(<').replace(')', '>)')
    return d



if __name__ == "__main__":
    main()
