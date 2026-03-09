#!/usr/bin/env bash
set -euo pipefail

usage() {
  echo "Usage: $0 <version>"
  echo "       $0 --rollback <version>"
  echo ""
  echo "  version      Semver version to release (e.g., 1.0.0 or v1.0.0)"
  echo "  --rollback   Clean up a failed release (delete tag and GitHub release)"
  echo ""
  echo "Release steps:"
  echo "  1. Validate the version and check prerequisites"
  echo "  2. Stamp COVERLINT_VERSION in action.yml and commit"
  echo "  3. Create and push a git tag"
  echo "  4. Wait for the SLSA release workflow to build and create the GitHub release"
  echo "  5. Update the major version tag (e.g., v1) for Actions marketplace"
  echo "  6. Update README.md examples with pinned commit SHAs"
  echo "  7. Reset COVERLINT_VERSION to dev placeholder"
  exit 1
}

if [[ $# -lt 1 ]]; then
  usage
fi

# --- Rollback mode ---
if [[ "$1" == "--rollback" ]]; then
  if [[ $# -ne 2 ]]; then
    echo "Usage: $0 --rollback <version>" >&2
    exit 1
  fi

  version="${2#v}"
  tag="v${version}"
  repo=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || echo "unknown")

  echo "Rollback plan for ${tag}:"

  # Check what exists
  local_tag=$(git rev-parse "$tag" 2>/dev/null || true)
  remote_tag=$(git ls-remote --tags origin "refs/tags/${tag}" 2>/dev/null | awk '{print $1}' || true)
  has_release=$(gh release view "$tag" --json tagName -q .tagName 2>/dev/null || true)

  [[ -n "$local_tag" ]]  && echo "  Local tag:      ${local_tag:0:7}" || echo "  Local tag:      (none)"
  [[ -n "$remote_tag" ]] && echo "  Remote tag:     ${remote_tag:0:7}" || echo "  Remote tag:     (none)"
  [[ -n "$has_release" ]] && echo "  GitHub release: yes" || echo "  GitHub release: (none)"

  if [[ -z "$local_tag" && -z "$remote_tag" && -z "$has_release" ]]; then
    echo ""
    echo "Nothing to roll back."
    exit 0
  fi

  echo ""
  read -rp "Delete all of the above? [y/N] " confirm
  if [[ "$confirm" != [yY] ]]; then
    echo "Aborted."
    exit 0
  fi

  if [[ -n "$has_release" ]]; then
    echo "Deleting GitHub release ${tag}..."
    gh release delete "$tag" --yes --cleanup-tag
  fi

  if [[ -n "$remote_tag" ]]; then
    echo "Deleting remote tag ${tag}..."
    git push origin ":refs/tags/${tag}" 2>/dev/null || true
  fi

  if [[ -n "$local_tag" ]]; then
    echo "Deleting local tag ${tag}..."
    git tag -d "$tag" 2>/dev/null || true
  fi

  echo ""
  echo "Rollback complete. You can now re-run: $0 ${version}"
  exit 0
fi

# --- Release mode ---
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
  echo ""
  echo "If the previous release failed, clean up first:"
  echo "  $0 --rollback ${version}"
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

# Stamp COVERLINT_VERSION in action.yml so SHA-pinned usage can resolve the release tag
echo ""
echo "Stamping COVERLINT_VERSION=${tag} in action.yml..."
perl -pi -e "s{COVERLINT_VERSION: \"[^\"]*\"}{COVERLINT_VERSION: \"${tag}\"}g" action.yml
git add action.yml
if ! git diff --cached --quiet; then
  git commit -m "Stamp version ${tag} in action.yml"
  git push origin "${branch}"
fi

# Create and push the version tag
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
  echo ""
  echo "To clean up and retry: $0 --rollback ${version}"
  exit 1
fi

echo "Workflow run: ${run_id}"
if ! gh run watch "$run_id" --exit-status; then
  echo "Error: release workflow failed" >&2
  echo "Check: gh run view ${run_id} --log-failed" >&2
  echo ""
  echo "To clean up and retry: $0 --rollback ${version}"
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

# Reset COVERLINT_VERSION to dev placeholder for the working branch
perl -pi -e "s{COVERLINT_VERSION: \"[^\"]*\"}{COVERLINT_VERSION: \"v0.0.0-dev\"}g" action.yml

git add README.md action.yml
if ! git diff --cached --quiet; then
  git commit -m "Post-release ${tag}: pin README SHAs, reset dev version"
  git push origin "${branch}"
fi

echo ""
echo "Done! ${tag} is live."
echo "  Release:     https://github.com/${repo}/releases/tag/${tag}"
echo "  Marketplace: https://github.com/marketplace/actions/coverlint"
echo ""
echo "Pin to this exact release in workflows:"
echo "  uses: evansims/coverlint@${commit_sha} # ${tag}"
