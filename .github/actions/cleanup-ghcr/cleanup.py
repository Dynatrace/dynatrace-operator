#!/usr/bin/env python3

# This script cleans up unreferenced package versions from GitHub Container Registry (ghcr.io).
# It keeps all versions that are referenced by tags updated within the last N days, as well
# as any versions matching specified regex patterns. It can be run in dry-run mode to preview deletions.

# It follows referenced digests starting from tagged versions, including multi-arch image manifests and Helm chart signatures
# This ensures that we don't accidentally delete any versions that are still in use, while allowing us to clean up old unreferenced versions.

import base64
import os
import requests
import re

# Read configuration from environment variables
ORG = os.environ.get('ORG', 'dynatrace')
PACKAGE = os.environ.get('PACKAGE', 'dynatrace-operator')
GITHUB_TOKEN = os.environ.get('GITHUB_TOKEN', '')
# base64-encoded token for ghcr.io, can be the same as GITHUB_TOKEN
GHCR_TOKEN = base64.b64encode(GITHUB_TOKEN.encode()).decode() if GITHUB_TOKEN else ''
DRY_RUN = os.environ.get('DRY_RUN', 'true').lower() in ('true', '1', 'yes')
PACKAGE_REPO_TYPE = os.environ.get('PACKAGE_REPO_TYPE', 'orgs')
KEEP_OUTDATED_TAGS_FOR_DAYS = int(os.environ.get('KEEP_OUTDATED_TAGS_FOR_DAYS', '14'))

# Parse comma-separated regex patterns
tags_to_keep_str = os.environ.get('TAGS_TO_ALWAYS_KEEP', '^snapshot$,^snapshot-release-.*')
TAGS_TO_ALWAYS_KEEP = [pattern.strip() for pattern in tags_to_keep_str.split(',')]

headers = {
    "Accept": "application/vnd.github+json",
    "Authorization": f"Bearer {GITHUB_TOKEN}",
    "X-GitHub-Api-Version": "2022-11-28"
}

def fetch_all_pages(url):
    """Fetch all pages from GitHub API."""
    data = []
    while url:
        resp = requests.get(url, headers=headers)
        resp.raise_for_status()
        data.extend(resp.json())

        # Get next page URL from Link header
        link = resp.headers.get('Link', '')
        url = None
        for part in link.split(','):
            if 'rel="next"' in part:
                url = part.split(';')[0].strip()[1:-1]
    return data

def fetch_manifest(tag):
    """Fetch manifest from ghcr.io."""
    url = f"https://ghcr.io/v2/{ORG}/{PACKAGE}/manifests/{tag}"
    try:
        resp = requests.get(url, headers={
            "Authorization": f"Bearer {GHCR_TOKEN}",
            "Accept": "application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.list.v2+json, application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json"
        })
        resp.raise_for_status()
        return resp.json(), resp.headers.get('docker-content-digest', '')
    except requests.HTTPError as e:
        if resp.status_code == 404:
            print(f"[WARN] Manifest not found for tag {tag} (404)")
            return None, ''
        else:
            print(f"[ERROR] Failed to fetch manifest for tag {tag}: {e}")
            return None, ''

# 1. Fetch all package versions
print(f"Fetching versions for {ORG}/{PACKAGE}...")
packages = fetch_all_pages(f"https://api.github.com/{PACKAGE_REPO_TYPE}/{ORG}/packages/container/{PACKAGE}/versions?per_page=100")
print(f"Found {len(packages)} packages")

# 2. Find referenced digests (from tagged versions)
print(f"Keeping all packages that have tags younger than {KEEP_OUTDATED_TAGS_FOR_DAYS} days...")
print(f"Exception: Always keep tags matching: {TAGS_TO_ALWAYS_KEEP}")

references_to_keep = set()
multiarch_image_tags_to_keep = set()
helm_tags_to_keep = set() # Helm packages have different dependencies to their signatures, so we track them separately

tagged_versions = [v for v in packages if v.get('metadata', {}).get('container', {}).get('tags')]

from datetime import datetime, timedelta, timezone
now = datetime.now(timezone.utc)
threshold_date = now - timedelta(days=KEEP_OUTDATED_TAGS_FOR_DAYS)

# Filter tags to only those updated within last N days
for v in tagged_versions:
    updated_at = v.get('updated_at') or v.get('created_at')
    if updated_at:
        updated_dt = datetime.strptime(updated_at, '%Y-%m-%dT%H:%M:%SZ').replace(tzinfo=timezone.utc)
        if updated_dt >= threshold_date:
            description = v.get('description', '').lower()
            if 'helm' in description:
                helm_tags_to_keep.update(v['metadata']['container']['tags'])
            else:
                multiarch_image_tags_to_keep.update(v['metadata']['container']['tags'])

    # Check if any tag matches any always-keep regex
    for t in v['metadata']['container']['tags']:
        if any(re.match(pattern, t) for pattern in TAGS_TO_ALWAYS_KEEP):
            print(f"Keeping {v['name']} because tag '{t}' matches always-keep pattern")
            description = v.get('description', '').lower()
            if 'helm' in description:
                helm_tags_to_keep.update(v['metadata']['container']['tags'])
            else:
                multiarch_image_tags_to_keep.update(v['metadata']['container']['tags'])
            break

# 3. Fetch manifests to get multi-arch digests
print("Keeping all digests referenced by those tags...")

print("  Tracing back multi arch images...")
for tag in multiarch_image_tags_to_keep:
    manifest, digest = fetch_manifest(tag)
    references_to_keep.add(digest)

    if manifest and 'manifests' in manifest:
        for m in manifest['manifests']:
            references_to_keep.add(m['digest'])

print("  Tracing back helm package signatures...")
for tag in helm_tags_to_keep:
    _, digest_of_helm_chart = fetch_manifest(tag)
    references_to_keep.add(digest_of_helm_chart)

    # replace : with - in tag to get the signature tag
    tag_of_reference_file = digest_of_helm_chart.replace(':', '-')
    manifest, digest_of_signature_reference_file = fetch_manifest(tag_of_reference_file)
    if manifest and digest_of_signature_reference_file:
        references_to_keep.add(digest_of_signature_reference_file)

        # add references for signature files
        for m in manifest.get('manifests', []):
            if 'digest' in m:
                references_to_keep.add(m['digest'])

print(f"Found {len(references_to_keep)} referenced digests")

# 4. Delete unreferenced versions
print(f"\nStarting deletion of unreferenced packages...")
deleted = 0
for v in packages:
    if v['name'] not in references_to_keep:
        print(f"{'[DRY-RUN]' if DRY_RUN else 'Deleting'} {v['name']}")
        if not DRY_RUN:
            resp = requests.delete(f"https://api.github.com/{PACKAGE_REPO_TYPE}/{ORG}/packages/container/{PACKAGE}/versions/{v['id']}", headers=headers)
            if resp.status_code == 204:
                print(f"Deleted {v['name']}")
            else:
                print(f"Failed to delete {v['name']}: {resp.status_code} {resp.text}")

        deleted += 1

print(f"\nTotal: {len(packages)}, Kept: {len(packages)-deleted}, Deleted: {deleted}")
if DRY_RUN:
    print("DRY-RUN mode - set DRY_RUN=False to actually delete")
