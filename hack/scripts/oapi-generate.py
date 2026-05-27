import sys
import os
import shutil
import subprocess
import tempfile
import urllib.request
import urllib.error
import yaml

SYNC_CONFIG = "api/oapi/sync-config.yaml"
GENERATOR_CONFIG = "api/oapi/generator-config.yaml"
IGNORE_FILE = "api/oapi/.openapi-generator-ignore"
GENERATOR_CLI = os.environ.get("OPENAPI_GENERATOR_CLI", "node_modules/.bin/openapi-generator-cli")


def find_all_refs(obj):
    """Recursively collect all $ref values from a nested dict/list structure.

    >>> find_all_refs({"$ref": "#/components/schemas/Foo"})
    ['#/components/schemas/Foo']
    >>> find_all_refs({"a": {"b": [{"$ref": "#/components/schemas/Bar"}, {"c": {"$ref": "#/components/schemas/Baz"}}]}})
    ['#/components/schemas/Bar', '#/components/schemas/Baz']
    >>> find_all_refs([])
    []
    """
    refs = []
    if isinstance(obj, dict):
        if "$ref" in obj:
            refs.append(obj["$ref"])
        for v in obj.values():
            refs.extend(find_all_refs(v))
    elif isinstance(obj, list):
        for item in obj:
            refs.extend(find_all_refs(item))
    return refs


def refs_in(spec, ref):
    """Return all $ref values within the part of spec pointed to by a JSON pointer ref.

    >>> refs_in({"components": {"schemas": {"Foo": {"$ref": "#/components/schemas/Bar"}}}}, "#/components/schemas/Foo")
    ['#/components/schemas/Bar']
    >>> refs_in({}, "external/ref")
    []
    >>> refs_in({}, "#/missing/path")
    []
    """
    if not ref.startswith("#/"):
        return []
    parts = ref[2:].split("/")
    node = spec
    for part in parts:
        if not isinstance(node, dict):
            return []
        node = node.get(part, {})
    return find_all_refs(node)


def download_spec(url, auth_token=None):
    req = urllib.request.Request(url)
    if auth_token:
        req.add_header("Authorization", f"Bearer {auth_token}")
    with urllib.request.urlopen(req) as resp:
        return resp.read()


def resolve_models(spec, api_tag):
    """Return sorted model names reachable from operations tagged with api_tag.

    >>> spec = {
    ...     "paths": {
    ...         "/foo": {"get": {"tags": ["FooApi"], "responses": {"200": {"$ref": "#/components/schemas/Foo"}}}}
    ...     },
    ...     "components": {"schemas": {"Foo": {}}},
    ... }
    >>> resolve_models(spec, "FooApi")
    ['Foo']
    >>> resolve_models(spec, "BarApi")
    []
    """
    worklist = []
    for path_item in spec.get("paths", {}).values():
        if not isinstance(path_item, dict):
            continue
        for operation in path_item.values():
            if not isinstance(operation, dict):
                continue
            if api_tag in operation.get("tags", []):
                for ref in find_all_refs(operation):
                    if ref not in worklist:
                        worklist.append(ref)
    for ref in worklist:
        for new_ref in refs_in(spec, ref):
            if new_ref not in worklist:
                worklist.append(new_ref)
    prefix = "#/components/schemas/"
    return sorted(ref[len(prefix):] for ref in worklist if ref.startswith(prefix))


def build_global_props(schema, default_global_props, spec):
    """Build the --global-property string for the generator.

    >>> build_global_props({}, "skipFormModel=true", {})
    'skipFormModel=true,apis,models'
    >>> build_global_props({"generate": {"globalProperties": {"apis": ["FooApi"]}}}, "", {"paths": {}, "components": {"schemas": {}}})
    'apis=FooApi'
    """
    gp = schema.get("generate", {}).get("globalProperties", {})
    apis = gp.get("apis", [])
    models = sorted({m for api in apis for m in resolve_models(spec, api)})

    parts = []
    if models:
        parts.append("models=" + ":".join(models))
    if default_global_props:
        parts.append(default_global_props)
    parts.append("apis=" + ":".join(apis) if apis else "apis,models")
    if additional := gp.get("additional"):
        parts.append(additional)

    return ",".join(parts)


def generate_schema(schema, defaults):
    name = schema.get("name")
    if not name:
        sys.exit("ERROR: schema entry is missing required field 'name'")
    gen = schema.get("generate", {})
    pkg = gen.get("packageName", name)
    version = gen.get("generatorVersion", defaults["version"]).lstrip("v")
    schema_additional_props = gen.get("additionalProperties", "")
    additional_props = ",".join(p for p in [defaults["additional"], schema_additional_props] if p)
    output_dir = f"{defaults['output_dir']}/{pkg}"

    spec_url = os.environ.get(schema.get("specUrlEnvVar", ""))
    if not spec_url:
        print(f"WARNING: no specUrlEnvVar set for {name}, skipping.")
        return

    auth_token = os.environ.get(schema.get("authEnvVar", ""))

    print(f"Downloading spec for {name}...")
    try:
        spec_data = download_spec(spec_url, auth_token)
    except urllib.error.URLError as e:
        sys.exit(f"ERROR: failed to download spec from {spec_url}: {e}")

    spec = yaml.safe_load(spec_data)

    with tempfile.NamedTemporaryFile(delete=False, suffix=".json") as tmp:
        tmp.write(spec_data)
        tmp_spec = tmp.name

    try:
        global_props = build_global_props(schema, defaults["global"], spec)

        print(f"Generating {name}, package: {pkg}...")
        shutil.rmtree(output_dir, ignore_errors=True)
        os.makedirs(output_dir, exist_ok=True)
        shutil.copy(IGNORE_FILE, f"{output_dir}/.openapi-generator-ignore")

        cmd = [
            GENERATOR_CLI, "generate",
            "-i", tmp_spec,
            "-g", "go",
            "-o", output_dir,
            "--package-name", pkg,
            "--skip-validate-spec",
            "--minimal-update",
        ]

        if additional_props:
            cmd += [f"--additional-properties={additional_props}"]
        if global_props:
            cmd += [f"--global-property={global_props}"]

        env = {**os.environ, "OPENAPI_GENERATOR_VERSION": version}
        if subprocess.run(cmd, env=env).returncode != 0:
            sys.exit(f"ERROR: generation failed for {name}")

        os.unlink(f"{output_dir}/.openapi-generator-ignore")
        print(f"Done: {output_dir}")
    finally:
        os.unlink(tmp_spec)


def main():
    with open(GENERATOR_CONFIG) as f:
        gen_cfg = yaml.safe_load(f)

    defaults = {
        "version": gen_cfg.get("generatorVersion", ""),
        "output_dir": gen_cfg.get("outputDir", ""),
        "additional": gen_cfg.get("additionalProperties", ""),
        "global": gen_cfg.get("globalProperties", ""),
    }

    with open(SYNC_CONFIG) as f:
        sync_cfg = yaml.safe_load(f)

    for schema in sync_cfg.get("schemas", []):
        generate_schema(schema, defaults)

    if os.path.exists("openapitools.json"):
        os.unlink("openapitools.json")


if __name__ == "__main__":
    main()
