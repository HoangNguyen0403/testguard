#!/usr/bin/env bash
set -euo pipefail

BUMP_TYPE="${1:-patch}"

# 1. Fetch latest tags
git fetch --tags --quiet || true

# 2. Get current latest tag across repo (using semver sort) or default
CURRENT_TAG="$(git tag -l "v[0-9]*" --sort=-v:refname 2>/dev/null | head -n1)"
CURRENT_TAG="${CURRENT_TAG:-v0.1.0}"
CLEAN_TAG="${CURRENT_TAG#v}"

IFS='.' read -r MAJOR MINOR PATCH <<< "${CLEAN_TAG}"
MAJOR="${MAJOR:-0}"
MINOR="${MINOR:-1}"
PATCH="${PATCH:-0}"

# 3. Compute next version
case "${BUMP_TYPE}" in
  patch)
    PATCH=$((PATCH + 1))
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  *)
    echo "❌ Unknown bump type: ${BUMP_TYPE}. Use: patch, minor, or major"
    exit 1
    ;;
esac

NEXT_TAG="v${MAJOR}.${MINOR}.${PATCH}"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 TestGuard Quick Release Manager"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Current Version: ${CURRENT_TAG}"
echo "Next Version   : ${NEXT_TAG} (${BUMP_TYPE} bump)"
echo ""
# 4. Check branch recommendation
CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")"
if [[ -n "${CURRENT_BRANCH}" && "${CURRENT_BRANCH}" != "main" && "${CURRENT_BRANCH}" != "master" ]]; then
  echo "⚠️ Warning: You are on branch '${CURRENT_BRANCH}'."
  echo "Releases should normally be tagged and published from 'main'."
  echo ""
fi

# 5. Check uncommitted changes
if ! git diff-index --quiet HEAD --; then
  echo "⚠️ Warning: You have uncommitted changes in your working tree."
  echo "Commit or stash changes before releasing."
  exit 1
fi

# 6. Pre-flight check: ensure target tag does not already exist
if git rev-parse -q --verify "refs/tags/${NEXT_TAG}" >/dev/null; then
  echo "❌ Error: Git tag '${NEXT_TAG}' already exists locally."
  echo "Existing tags: $(git tag -l "v[0-9]*" --sort=-v:refname | head -n5 | tr '\n' ' ')"
  exit 1
fi

if git ls-remote --tags origin "refs/tags/${NEXT_TAG}" 2>/dev/null | grep -q "${NEXT_TAG}"; then
  echo "❌ Error: Git tag '${NEXT_TAG}' already exists on remote origin."
  echo "Run 'git fetch --tags' to update your local tags."
  exit 1
fi

# 7. Tag and push
echo "🏷️ Creating git tag ${NEXT_TAG}..."
git tag -a "${NEXT_TAG}" -m "Release ${NEXT_TAG}"

echo "⬆️ Pushing tag ${NEXT_TAG} to origin..."
git push origin "${NEXT_TAG}"

echo ""
echo "🎉 Release ${NEXT_TAG} successfully triggered!"
echo "👉 GitHub Actions workflow: https://github.com/HoangNguyen0403/testguard/actions"
echo "👉 GitHub Releases        : https://github.com/HoangNguyen0403/testguard/releases/tag/${NEXT_TAG}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
