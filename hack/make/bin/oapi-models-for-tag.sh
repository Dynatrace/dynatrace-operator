#!/usr/bin/env bash
# Usage: oapi-models-for-tag.sh <spec-file> <tag>
# Prints all model names (direct + transitive) used by operations with the given tag.
set -euo pipefail

SPEC="${1:?Usage: $0 <spec-file> <tag>}"
TAG="${2:?Usage: $0 <spec-file> <tag>}"

jq -r --arg tag "$TAG" '
  . as $spec |

  def refs_in($ref):
    ($ref | ltrimstr("#/") | split("/")) as $parts |
    if ($parts | length) == 3
    then $spec | getpath(["components", $parts[1], $parts[2]]) // {}
    else {} end |
    [.. | objects | select(has("$ref")) | .["$ref"]] | .[];

  ([.paths | to_entries[] |
    .value | to_entries[] |
    select(.value.tags? | index($tag)) |
    [.. | objects | select(has("$ref")) | .["$ref"]]
  ] | flatten | unique) as $seed |

  { queue: $seed,
    visited: ($seed | map({key: ., value: true}) | from_entries) } |

  until(.queue | length == 0;
    .queue[0] as $ref |
    . as $state |
    ([refs_in($ref)] | map(select($state.visited[.] == null))) as $new |
    { queue: ($state.queue[1:] + $new),
      visited: ($state.visited + ($new | map({key: ., value: true}) | from_entries)) }
  ) |

  .visited | keys |
  map(select(startswith("#/components/schemas/"))) |
  map(ltrimstr("#/components/schemas/")) |
  sort | .[]
' "$SPEC"
