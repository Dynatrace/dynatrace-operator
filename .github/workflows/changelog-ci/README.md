## What is Changelog CI?

Changelog CI is a GitHub Action that enables a project to utilize an
automatically generated changelog.

The workflow can be configured to perform **any (or all)** of the following actions

* **Generate** changelog using **Pull Request** or **Commit Messages**.

* **Prepend** the generated changelog to the `CHANGELOG.md` file and then **Commit** modified `CHANGELOG.md` file to the release pull request.

* Add a **Comment** on the release pull request with the generated changelog.

## How Does It Work:

Changelog CI uses `python` and `GitHub API` to generate changelog for a
repository. First, it tries to get the `latest release` from the repository (If
available). Then, it checks all the **pull requests** / **commits** merged after the last release
using the GitHub API. After that, it parses the data and generates
the `changelog`. Finally, It writes the generated changelog at the beginning of
the `CHANGELOG.md` (or user-provided filename) file. In addition to that, if a
user provides a config (JSON/YAML file), Changelog CI parses the user-provided config
file and renders the changelog according to users config. Then the changes
are **committed** and/or **commented** to the release Pull request.

## Usage:

To use this Action The pull **request title** must match with the
default `regex`
or the user-provided `regex` from the config file.

**Default Title Regex:** `^(?i:release)` (title must start with the word "
release" (case-insensitive))

**Default Changelog Type:** `pull_request` (Changelog will be generated using pull request title),
You can generate changelog using `commit_message` as well
[Using an optional configuration file](#using-an-optional-configuration-file).

**Default Version Number Regex:** This Regex will be checked against a Pull
Request title. This follows [`SemVer`](https://regex101.com/r/Ly7O1x/3/) (
Semantic Versioning) pattern. e.g. `1.0.0`, `1.0`, `v1.0.1` etc.

For more details on **Semantic Versioning pattern** go to this
link: https://regex101.com/r/Ly7O1x/3/

**Note:** You can use a custom regular expression to parse your changelog adding
one to the optional configuration file. To learn more, see
[Using an optional configuration file](#using-an-optional-configuration-file).

## Configuration

### Using an optional configuration file

Changelog CI is will run perfectly fine without including a configuration file.
If a user seeks to modify the default behaviors of Changelog CI, they can do so
by adding a `JSON` or `YAML` config file to the project. For example:

* Including `JSON` file `changelog-ci-config.json`:

    ```yaml
    with:
      config_file: "changelog-ci-config.json"
    ```

* Including `YAML` file `changelog-ci-config.yaml`:

    ```yaml
    with:
      config_file: "changelog-ci-config.yml"
    ```

### Valid options

* `changelog_type`
  You can use `pull_request` (Default) or `commit_message` as the value for this option.
  `pull_request` option will generate changelog using pull request title.
  `commit_message` option will generate changelog using commit messages.

* `header_prefix`
  The prefix before the version number. e.g. `version:` in `Version: 1.0.2`

* `commit_changelog`
  Value can be `true` or `false`. if not provided defaults to `true`. If it is
  set to `true` then Changelog CI will commit to the release pull request.

* `pull_request_title_regex`
  If the pull request title matches with this `regex` Changelog CI will generate
  changelog for it. Otherwise, it will skip the changelog generation.
  If `pull_request_title_regex` is not provided defaults to `^(?i:release)`,
  then the title must begin with the word "release" (case-insensitive).

* `version_regex`
  This `regex` tries to find the version number from the pull request title. in
  case of no match, changelog generation will be skipped. if `version_regex` is
  not provided, it defaults to
  [`SemVer`](https://regex101.com/r/Ly7O1x/3/) pattern.

* `group_config`
  By adding this you can group changelog items by your repository labels with
  custom titles.

### Example Config File

Written in JSON:

```json
{
  "changelog_type": "commit_message",
  "header_prefix": "Version:",
  "commit_changelog": true,
  "pull_request_title_regex": "^Release",
  "version_regex": "v?([0-9]{1,2})+[.]+([0-9]{1,2})+[.]+([0-9]{1,2})\\s\\(\\d{1,2}-\\d{1,2}-\\d{4}\\)",
  "group_config": [
    {
      "title": "Bug Fixes",
      "labels": ["bug", "bugfix"]
    },
    {
      "title": "Code Improvements",
      "labels": ["improvements", "enhancement"]
    },
    {
      "title": "New Features",
      "labels": ["feature"]
    },
    {
      "title": "Documentation Updates",
      "labels": ["docs", "documentation", "doc"]
    }
  ]
}
```

Written in YAML:

```yaml
changelog_type: 'commit_message' # or 'pull_request'
header_prefix: 'Version:'
commit_changelog: true
pull_request_title_regex: '^Release'
version_regex: 'v?([0-9]{1,2})+[.]+([0-9]{1,2})+[.]+([0-9]{1,2})\s\(\d{1,2}-\d{1,2}-\d{4}\)'
group_config:
  - title: Bug Fixes
    labels:
      - bug
      - bugfix
  - title: Code Improvements
    labels:
      - improvements
      - enhancement
  - title: New Features
    labels:
      - feature
  - title: Documentation Updates
    labels:
      - docs
      - documentation
      - doc
```


* Here the changelog will be generated using commit messages because of `changelog_type: 'commit_message'`.

* Here **`pull_request_title_regex`** will match any pull request title that
starts with **`Release`**
you can match **Any Pull Request Title** by adding  this **`pull_request_title_regex": ".*"`**,


## Example Workflow

```yaml
name: Changelog CI

# Controls when the action will run. Triggers the workflow on a pull request
on:
  pull_request:
    types: [ opened, reopened ]

jobs:
  build:
    runs-on: ubuntu-latest

    steps:
      # Checks-out your repository
      - uses: actions/checkout@v2

      - name: Run Changelog CI
        uses: "./changelog-ci"
        with:
          changelog_filename: CHANGELOG.md
          config_file: "changelog-ci-config.json"
        # Add this if you are using it on a private repository
        # Or if you have turned on commenting through the config file.
        env:
          GITHUB_TOKEN: ${{secrets.GITHUB_TOKEN}}
```
# License

The code in this project is released under the [MIT License](LICENSE).
