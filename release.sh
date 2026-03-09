#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <version>"
  echo ""
  echo "  version   Semver version to release (e.g., 1.0.0 or v1.0.0)"
  echo ""
  echo "This script will:"
  echo "  1. Validate the version and check prerequisites"
  echo "  2. Create and push a git tag"
  echo "  3. Wait for the GoReleaser CI workflow to create the GitHub release"
  echo "  4. Update the major version tag (e.g., v1) for Actions marketplace"
  echo "  5. Update README.md examples with pinned commit SHAs (coverlint + third-party actions)"
  exit 1
}

if [[ $# -ne 1 ]]; then
  usage
fi

# Normalize version: strip leading 'v' then re-add it
version="${1#v}"
tag="v${version}"
major="v$(echo "$version" | cut -d. -f1)"
branch=$(git branch --show-current)

# Validate semver format
if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
  echo "Error: '${version}' is not a valid semver version (expected: X.Y.Z)" >&2
  exit 1
fi

# Check prerequisites
if ! command -v gh &>/dev/null; then
  echo "Error: gh CLI is required (https://cli.github.com)" >&2
  exit 1
fi

if ! git diff --quiet HEAD 2>/dev/null; then
  echo "Error: working tree has uncommitted changes" >&2
  exit 1
fi

if git rev-parse "$tag" &>/dev/null; then
  echo "Error: tag '${tag}' already exists" >&2
  exit 1
fi

# Confirm
echo "Release plan:"
echo "  Tag:           ${tag}"
echo "  Major tag:     ${major}"
echo "  Branch:        ${branch}"
echo "  Commit:        $(git rev-parse --short HEAD)"
echo ""
read -rp "Proceed? [y/N] " confirm
if [[ "$confirm" != [yY] ]]; then
  echo "Aborted."
  exit 0
fi

# Create and push the version tag
echo ""
echo "Creating tag ${tag}..."
git tag "$tag"
git push origin "$tag"

# Wait for the release workflow to complete
echo "Waiting for release workflow..."
run_id=""
for i in $(seq 1 30); do
  run_id=$(gh run list --workflow=release.yml --limit=1 --json databaseId,headBranch --jq ".[] | select(.headBranch == \"${tag}\") | .databaseId" 2>/dev/null || true)
  if [[ -n "$run_id" ]]; then
    break
  fi
  sleep 2
done

if [[ -z "$run_id" ]]; then
  echo "Error: could not find release workflow run for ${tag}" >&2
  echo "Check https://github.com/$(gh repo view --json nameWithOwner -q .nameWithOwner)/actions/workflows/release.yml" >&2
  exit 1
fi

echo "Workflow run: ${run_id}"
if ! gh run watch "$run_id" --exit-status; then
  echo "Error: release workflow failed" >&2
  echo "Check: gh run view ${run_id} --log-failed" >&2
  exit 1
fi

echo ""
echo "Release ${tag} created successfully."

# Update major version tag
echo "Updating ${major} tag..."
git tag -fa "$major" -m "Update ${major} tag to ${tag}"
git push origin "$major" --force

commit_sha=$(git rev-parse "$tag")
repo=$(gh repo view --json nameWithOwner -q .nameWithOwner)

# Append SHA pinning guidance to the GitHub Release body
existing_body=$(gh release view "$tag" --json body -q .body 2>/dev/null || true)
pin_section="## SHA Pinning

For security hardening, pin to the exact commit SHA for this release:

\`\`\`yaml
- uses: evansims/coverlint@${commit_sha} # ${tag}
\`\`\`

See [GitHub's guide on security hardening](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions#using-third-party-actions) for details."

gh release edit "$tag" --notes "${existing_body}

${pin_section}"

# Resolve a git tag to its commit SHA, dereferencing annotated tags
resolve_tag_sha() {
  local repo="$1" tag="$2"
  local ref_json sha obj_type

  ref_json=$(gh api "repos/${repo}/git/ref/tags/${tag}" 2>/dev/null) || return 1
  sha=$(echo "$ref_json" | jq -r '.object.sha')
  obj_type=$(echo "$ref_json" | jq -r '.object.type')

  if [[ "$obj_type" == "tag" ]]; then
    sha=$(gh api "repos/${repo}/git/tags/${sha}" --jq '.object.sha' 2>/dev/null) || return 1
  fi

  echo "$sha"
}

# Update README.md usage examples with pinned SHA
echo "Updating README.md with pinned SHA..."
perl -pi -e "s{uses: evansims/coverlint\@\S+(\s+#\s*\S+)?}{uses: evansims/coverlint\@${commit_sha} # ${tag}}g" README.md

# Update third-party action SHAs to their latest release commits
echo "Resolving third-party action SHAs..."
perl -ne 'print "$1 $2\n" if m{uses:\s+(?!evansims/coverlint)(\S+?)\@\S+\s+#\s*(\S+)}' README.md | sort -u | while IFS=' ' read -r action version_tag; do
  [[ -z "$action" ]] && continue

  # Extract the repo (first two path components, e.g. github/codeql-action from github/codeql-action/upload-sarif)
  action_repo=$(echo "$action" | cut -d/ -f1,2)

  action_sha=$(resolve_tag_sha "$action_repo" "$version_tag") || {
    echo "  Warning: could not resolve ${action_repo}@${version_tag}, skipping" >&2
    continue
  }

  echo "  ${action}@${version_tag} → ${action_sha:0:7}"
  perl -pi -e "s{uses: \Q${action}\E\@\S+(\s+#\s*\S+)?}{uses: ${action}\@${action_sha} # ${version_tag}}g" README.md
done

git add README.md
if ! git diff --cached --quiet; then
  git commit -m "Pin README examples to ${tag} (${commit_sha:0:7})"
  git push origin "${branch}"
fi

echo ""
echo "Done! ${tag} is live."
echo "  Release:     https://github.com/${repo}/releases/tag/${tag}"
echo "  Marketplace: https://github.com/marketplace/actions/coverlint"
echo ""
echo "Pin to this exact release in workflows:"
echo "  uses: evansims/coverlint@${commit_sha} # ${tag}"
