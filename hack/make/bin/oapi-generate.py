#!/usr/bin/env python3
"""Generate Go SDKs from OpenAPI specs defined in sync-config.yaml."""

import sys
import os
import shutil
import subprocess
import tempfile
import urllib.request
import urllib.error
import yaml
from pathlib import Path

SYNC_CONFIG = "api/oapi/sync-config.yaml"
GENERATOR_CONFIG = "api/oapi/generator-config.yaml"
IGNORE_FILE = "api/oapi/.openapi-generator-ignore"
GENERATOR_CLI = os.environ.get("OPENAPI_GENERATOR_CLI", "node_modules/.bin/openapi-generator-cli")


def find_all_refs(obj):
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
    try:
        with urllib.request.urlopen(req) as resp:
            return resp.read()
    except urllib.error.URLError as e:
        raise RuntimeError(f"failed to download spec from {url}: {e}")


def resolve_models(spec, api_tag):
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


def build_global_props(schema, default_global_props, models):
    gp = (schema.get("generate") or {}).get("globalProperties") or {}
    apis = gp.get("apis") or []

    parts = []
    if models:
        parts.append("models=" + ":".join(models))
    if default_global_props:
        parts.append(default_global_props)
    parts.append("apis=" + ":".join(apis) if apis else "apis,models")
    if additional := gp.get("additional"):
        parts.append(additional)

    return ",".join(parts)


def main():
    with open(GENERATOR_CONFIG) as f:
        gen_cfg = yaml.safe_load(f)

    default_version = gen_cfg.get("generatorVersion", "").lstrip("v")
    default_output_dir = gen_cfg.get("outputDir", "")
    default_additional_props = gen_cfg.get("additionalProperties", "")
    default_global_props = gen_cfg.get("globalProperties", "")

    with open(SYNC_CONFIG) as f:
        sync_cfg = yaml.safe_load(f)

    for schema in sync_cfg.get("schemas") or []:
        name = schema["name"]
        pkg = (schema.get("generate") or {}).get("packageName") or name
        version = ((schema.get("generate") or {}).get("generatorVersion") or default_version).lstrip("v")
        schema_additional_props = (schema.get("generate") or {}).get("additionalProperties") or ""
        additional_props = ",".join(filter(None, [default_additional_props, schema_additional_props]))
        output_dir = f"{default_output_dir}/{pkg}"

        spec_url_var = schema.get("specUrlEnvVar", "")
        spec_url = os.environ.get(spec_url_var, "") if spec_url_var else ""
        if not spec_url:
            print(f"WARNING: no specUrlEnvVar set for {name}, skipping.")
            continue

        auth_var = schema.get("authEnvVar", "")
        auth_token = os.environ.get(auth_var, "") if auth_var else ""

        print(f"Downloading spec for {name}...")
        try:
            spec_data = download_spec(spec_url, auth_token or None)
        except RuntimeError as e:
            print(f"ERROR: {e}", file=sys.stderr)
            sys.exit(1)

        spec = yaml.safe_load(spec_data)
        apis = ((schema.get("generate") or {}).get("globalProperties") or {}).get("apis") or []
        models = sorted({m for api in apis for m in resolve_models(spec, api)})

        with tempfile.NamedTemporaryFile(delete=False, suffix=".json") as tmp:
            tmp.write(spec_data)
            tmp_spec = tmp.name

        try:
            global_props = build_global_props(schema, default_global_props, models)

            print(f"Generating {name}, package: {pkg}...")
            shutil.rmtree(output_dir, ignore_errors=True)
            Path(output_dir).mkdir(parents=True, exist_ok=True)
            shutil.copy(IGNORE_FILE, f"{output_dir}/.openapi-generator-ignore")

            cmd = [
                GENERATOR_CLI, "generate",
                "-i", tmp_spec, "-g", "go", "-o", output_dir,
                "--skip-validate-spec", "--minimal-update",
            ]
            if pkg:
                cmd += ["--package-name", pkg]
            if additional_props:
                cmd += [f"--additional-properties={additional_props}"]
            if global_props:
                cmd += [f"--global-property={global_props}"]
            if auth_token:
                cmd += ["--auth", f"Authorization:Bearer%20{auth_token}"]

            env = {**os.environ, "OPENAPI_GENERATOR_VERSION": version}
            if subprocess.run(cmd, env=env).returncode != 0:
                print(f"ERROR: generation failed for {name}", file=sys.stderr)
                sys.exit(1)

            os.unlink(f"{output_dir}/.openapi-generator-ignore")
            print(f"Done: {output_dir}")
        finally:
            os.unlink(tmp_spec)

    Path("openapitools.json").unlink(missing_ok=True)


if __name__ == "__main__":
    main()
