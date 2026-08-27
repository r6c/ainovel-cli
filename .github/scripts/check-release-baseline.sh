#!/bin/sh
set -eu

REPO=${1:-.}
TAG=${RELEASE_TAG:-}
EXPECTED_SHA=${RELEASE_SHA:-}
NOTES=${RELEASE_NOTES:-}

fail() {
    echo "release baseline check failed: $*" >&2
    exit 1
}

[ -n "$TAG" ] || fail "RELEASE_TAG is required"
[ -n "$EXPECTED_SHA" ] || fail "RELEASE_SHA is required"
[ -n "$NOTES" ] || fail "RELEASE_NOTES is required"
[ -f "$NOTES" ] || fail "release notes not found: $NOTES"

CURRENT_SHA=$(git -C "$REPO" rev-parse HEAD 2>/dev/null) || fail "not a Git worktree: $REPO"
TAG_SHA=$(git -C "$REPO" rev-parse "${TAG}^{commit}" 2>/dev/null) || fail "release tag not found: $TAG"

[ "$CURRENT_SHA" = "$EXPECTED_SHA" ] || fail "checkout $CURRENT_SHA does not match RELEASE_SHA $EXPECTED_SHA"
[ "$TAG_SHA" = "$EXPECTED_SHA" ] || fail "tag $TAG points to $TAG_SHA, expected $EXPECTED_SHA"

[ "$(sed -n '1p' "$NOTES")" = "<!-- release-tag: $TAG -->" ] || fail "release notes header does not identify tag $TAG"
[ "$(sed -n '2p' "$NOTES")" = "<!-- release-sha: $EXPECTED_SHA -->" ] || fail "release notes header does not identify commit $EXPECTED_SHA"

echo "release baseline ok: $TAG ($EXPECTED_SHA)"
