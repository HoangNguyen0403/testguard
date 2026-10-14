#!/usr/bin/env bash
set -euo pipefail

BUMP_TYPE="${1:-patch}"

# 1. Fetch latest tags
git fetch --tags --quiet || true

# 2. Get current latest tag or default
CURRENT_TAG="$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")"
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

# 4. Check uncommitted changes
if ! git diff-index --quiet HEAD --; then
  echo "⚠️ Warning: You have uncommitted changes in your working tree."
  echo "Commit or stash changes before releasing."
  exit 1
fi

# 5. Tag and push
echo "🏷️ Creating git tag ${NEXT_TAG}..."
git tag -a "${NEXT_TAG}" -m "Release ${NEXT_TAG}"

echo "⬆️ Pushing tag ${NEXT_TAG} to origin..."
git push origin "${NEXT_TAG}"

echo ""
echo "🎉 Release ${NEXT_TAG} successfully triggered!"
echo "👉 GitHub Actions workflow: https://github.com/HoangNguyen0403/testguard/actions"
echo "👉 GitHub Releases        : https://github.com/HoangNguyen0403/testguard/releases/tag/${NEXT_TAG}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
