# Generating an OpenAPI client

Generated clients land in `pkg/clients/generated/<packageName>` via the **Sync OpenAPI schemas** GitHub Action.

Adding a new client requires changes in three places: GitHub secrets, the workflow file, and `sync-config.yaml`.

## 1. Get a REST API spec URL

The URL must point to a REST API endpoint that serves the raw spec and accepts a Bearer token in the `Authorization` header. Every major provider (Bitbucket, GitHub, GitLab, …) exposes one, check their docs for the exact path.

Browser "raw" links usually don't work, they rely on session cookies, not Bearer tokens.

## 2. Add secrets in GitHub

Secrets are stored in the **`oapi` environment** (repo Settings → *Environments* → `oapi`):

- `<NAME>_SPEC_URL` — the URL from step 1
- `<NAME>_TOKEN` — Bearer token (skip for public specs)

## 3. Wire the secrets into the workflow

Edit `.github/workflows/sync-openapi-schema.yaml`, in the `sync-schemas` job, add to `env:`:

```yaml
env:
  MY_API_SPEC_URL:        ${{ secrets.MY_API_SPEC_URL }}
  MY_API_TOKEN:           ${{ secrets.MY_API_TOKEN }}
```

## 4. Add a schema entry to `api/oapi/sync-config.yaml`

The full field list and a structure example are documented as comments at the top of `api/oapi/sync-config.yaml`.
Example:

```yaml
schemas:
  - name: myApi
    specUrlEnvVar: "MY_API_SPEC_URL" # Passed through the action job from previous step
    authEnvVar: "MY_API_TOKEN" # same as specUrlEnvVar
    generate:
      packageName: myapi
      globalProperties:
        apis:
          - Users # Models for the api will be autoresolved
```

Triggering the workflow now produces the clients at `pkg/clients/generated/<packageName>/`.

## 5. Tweak ignored files (optional)

`api/oapi/.openapi-generator-ignore` lists files the generator should skip (READMEs, tests, `.travis.yml`, etc.). It applies to **every** schema. Add patterns there if you want to drop more.

## 6. Cross-schema defaults (optional)

`api/oapi/generator-config.yaml` holds the defaults that apply to every schema unless overridden per-schema:

```yaml
generatorVersion: v7.22.0
outputDir: pkg/clients/generated
additionalProperties: "generateInterfaces"
globalProperties: "supportingFiles"
```

Edit it to change versions or properties for all clients at once.

## 7. Run the workflow

GitHub → *Actions* → **Sync OpenAPI schemas** → *Run workflow*.

It downloads each spec, regenerates the clients, and opens a PR with the diff under `pkg/clients/generated/`. Merge to apply.

## Local generation (optional)

```sh
export MY_API_SPEC_URL="..."
export MY_API_TOKEN="..."
make oapi/generate
```
