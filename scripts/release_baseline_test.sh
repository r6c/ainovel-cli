#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
CHECKER="$ROOT/.github/scripts/check-release-baseline.sh"
WORKFLOW="$ROOT/.github/workflows/release.yml"
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

fail() {
    echo "release baseline test failed: $*" >&2
    exit 1
}

# The workflow must pass the tag and commit explicitly and verify them before GoReleaser.
grep -F 'RELEASE_TAG: ${{ github.ref_name }}' "$WORKFLOW" >/dev/null || fail "workflow does not pass RELEASE_TAG"
grep -F 'RELEASE_SHA: ${{ github.sha }}' "$WORKFLOW" >/dev/null || fail "workflow does not pass RELEASE_SHA"
grep -F 'name: Verify release baseline' "$WORKFLOW" >/dev/null || fail "workflow has no baseline step"

run_checker() {
    RELEASE_TAG=$1 RELEASE_SHA=$2 RELEASE_NOTES=$TMP/release-notes.md \
        "$CHECKER" "$TMP/repo"
}

expect_failure() {
    if "$@" >/dev/null 2>&1; then
        fail "expected failure: $*"
    fi
}

mkdir "$TMP/repo"
git -C "$TMP/repo" init -q
git -C "$TMP/repo" config user.name test
git -C "$TMP/repo" config user.email test@example.invalid
printf '%s\n' baseline > "$TMP/repo/file.txt"
git -C "$TMP/repo" add file.txt
git -C "$TMP/repo" commit -q -m baseline
SHA=$(git -C "$TMP/repo" rev-parse HEAD)
git -C "$TMP/repo" tag v0.1.0-rc.2
git -C "$TMP/repo" tag v0.1.0

cat > "$TMP/release-notes.md" <<EOF
<!-- release-tag: v0.1.0 -->
<!-- release-sha: $SHA -->

## 变更

稳定版基线测试。
EOF

# Valid: the explicit release identity and notes metadata agree.
run_checker v0.1.0 "$SHA"

# A different explicit tag with notes for the stable release must fail closed.
expect_failure env RELEASE_TAG=v0.1.0-rc.2 RELEASE_SHA="$SHA" RELEASE_NOTES="$TMP/release-notes.md" "$CHECKER" "$TMP/repo"

# A detached/current checkout at a different commit must fail closed.
printf '%s\n' changed >> "$TMP/repo/file.txt"
git -C "$TMP/repo" add file.txt
git -C "$TMP/repo" commit -q -m changed
NEW_SHA=$(git -C "$TMP/repo" rev-parse HEAD)
expect_failure env RELEASE_TAG=v0.1.0 RELEASE_SHA="$SHA" RELEASE_NOTES="$TMP/release-notes.md" "$CHECKER" "$TMP/repo"

# Restore the tested commit and reject mismatched release-notes metadata.
git -C "$TMP/repo" checkout -q "$SHA"
cat > "$TMP/release-notes.md" <<EOF
<!-- release-tag: v0.1.0-rc.2 -->
<!-- release-sha: $SHA -->
EOF
expect_failure env RELEASE_TAG=v0.1.0 RELEASE_SHA="$SHA" RELEASE_NOTES="$TMP/release-notes.md" "$CHECKER" "$TMP/repo"

# Release notes generation must also use the explicit release identity.
NOTES_REPO="$TMP/notes-repo"
mkdir "$NOTES_REPO"
git -C "$NOTES_REPO" init -q
git -C "$NOTES_REPO" config user.name test
git -C "$NOTES_REPO" config user.email test@example.invalid
printf '%s\n' notes > "$NOTES_REPO/file.txt"
git -C "$NOTES_REPO" add file.txt
git -C "$NOTES_REPO" commit -q -m notes
NOTES_SHA=$(git -C "$NOTES_REPO" rev-parse HEAD)
git -C "$NOTES_REPO" tag v0.1.0-rc.2
git -C "$NOTES_REPO" tag v0.1.0

NOTES_OUTPUT=$(cd "$NOTES_REPO" && RELEASE_TAG=v0.1.0 RELEASE_SHA="$NOTES_SHA" sh "$ROOT/.github/scripts/gen-changelog.sh")
printf '%s\n' "$NOTES_OUTPUT" | grep -F "<!-- release-tag: v0.1.0 -->" >/dev/null || fail "release notes used the wrong tag"
printf '%s\n' "$NOTES_OUTPUT" | grep -F "<!-- release-sha: $NOTES_SHA -->" >/dev/null || fail "release notes omitted the release commit"

# Explicit identity must win even when the older RC tag points at the same commit.
if printf '%s\n' "$NOTES_OUTPUT" | grep -F "v0.1.0-rc.2" >/dev/null; then
    fail "release notes mentioned the older RC tag"
fi

# The baseline metadata must be the first two lines, not merely present somewhere in the body.
FIRST_LINES=$(printf '%s\n' "$NOTES_OUTPUT" | sed -n '1,2p')
EXPECTED_HEADER=$(printf '<!-- release-tag: v0.1.0 -->\n<!-- release-sha: %s -->' "$NOTES_SHA")
[ "$FIRST_LINES" = "$EXPECTED_HEADER" ] || fail "release metadata is not at the top"

# An empty range still carries the explicit release identity.
EMPTY_NOTES_REPO="$TMP/empty-notes-repo"
mkdir "$EMPTY_NOTES_REPO"
git -C "$EMPTY_NOTES_REPO" init -q
git -C "$EMPTY_NOTES_REPO" config user.name test
git -C "$EMPTY_NOTES_REPO" config user.email test@example.invalid
printf '%s\n' empty > "$EMPTY_NOTES_REPO/file.txt"
git -C "$EMPTY_NOTES_REPO" add file.txt
git -C "$EMPTY_NOTES_REPO" commit -q -m empty
EMPTY_SHA=$(git -C "$EMPTY_NOTES_REPO" rev-parse HEAD)
git -C "$EMPTY_NOTES_REPO" tag v0.2.0
EMPTY_OUTPUT=$(cd "$EMPTY_NOTES_REPO" && RELEASE_TAG=v0.2.0 RELEASE_SHA="$EMPTY_SHA" RELEASE_PREV_TAG=v0.2.0 sh "$ROOT/.github/scripts/gen-changelog.sh")
printf '%s\n' "$EMPTY_OUTPUT" | sed -n '1p' | grep -F '<!-- release-tag: v0.2.0 -->' >/dev/null || fail "empty release notes omitted tag"
printf '%s\n' "$EMPTY_OUTPUT" | sed -n '2p' | grep -F "<!-- release-sha: $EMPTY_SHA -->" >/dev/null || fail "empty release notes omitted sha"

# A successful HTTP response with malformed AI output must still fall back deterministically.
FAKE_BIN="$TMP/fake-bin"
mkdir "$FAKE_BIN"
cat > "$FAKE_BIN/curl" <<'EOF'
#!/bin/sh
set -eu
out=
while [ "$#" -gt 0 ]; do
    if [ "$1" = "-o" ]; then
        out=$2
        shift 2
    else
        shift
    fi
done
[ -n "$out" ]
printf '%s\n' '{"unexpected":true}' > "$out"
EOF
chmod +x "$FAKE_BIN/curl"
MALFORMED_OUTPUT=$(cd "$NOTES_REPO" && PATH="$FAKE_BIN:$PATH" GEMINI_API_KEY=test RELEASE_TAG=v0.1.0 RELEASE_SHA="$NOTES_SHA" sh "$ROOT/.github/scripts/gen-changelog.sh" 2>&1)
printf '%s\n' "$MALFORMED_OUTPUT" | sed -n '1p' | grep -F '<!-- release-tag: v0.1.0 -->' >/dev/null || fail "malformed AI response omitted tag"
printf '%s\n' "$MALFORMED_OUTPUT" | grep -F "## What's Changed" >/dev/null || fail "malformed AI response did not use fallback"
if printf '%s\n' "$MALFORMED_OUTPUT" | grep -E 'Traceback|KeyError|JSONDecodeError' >/dev/null; then
    fail "malformed AI response leaked parser diagnostics"
fi

# The baseline metadata must be emitted exactly once.
HEADER_COUNT=$(printf '%s\n' "$MALFORMED_OUTPUT" | grep -c '^<!-- release-tag:' || true)
[ "$HEADER_COUNT" -eq 1 ] || fail "fallback emitted duplicate release metadata"

echo "release baseline tests passed"
