#!/usr/bin/env bash
# Publishes staged parquet to the pages branch. Never fails the host job:
# every abnormal path ends in ::warning:: + exit 0.
set -uo pipefail

warn() { echo "::warning::testlake: $*"; }

main() {
  # Guard against unset required env vars: under `set -u`, referencing a
  # truly-unset variable aborts the whole process immediately (bypassing
  # any `||` handler), so validate with `${VAR:-}` before using it bare.
  if [ -z "${GITHUB_ACTION_PATH:-}" ]; then
    warn "GITHUB_ACTION_PATH not set"
    return 0
  fi
  if [ -z "${INPUT_REPORTS:-}" ]; then
    warn "INPUT_REPORTS not set"
    return 0
  fi
  local branch="${INPUT_BRANCH:-gh-pages}"
  local viewer="${INPUT_VIEWER:-false}"
  local retention="${INPUT_RETENTION_DAYS:-400}"

  local bin="${RUNNER_TEMP:-/tmp}/testlake-bin"
  ( cd "$GITHUB_ACTION_PATH/collector" && go build -o "$bin" ./cmd/testlake ) || { warn "go build failed"; return 0; }

  local staging
  staging="$(mktemp -d)" || { warn "mktemp failed"; return 0; }
  "$bin" collect --reports "$INPUT_REPORTS" --data "$staging" || { warn "collect failed"; return 0; }

  local remote="${TESTLAKE_REMOTE_URL:-https://x-access-token:${GITHUB_TOKEN:-}@github.com/${GITHUB_REPOSITORY:-}.git}"

  for attempt in 1 2 3 4 5; do
    local work
    work="$(mktemp -d)" || { warn "mktemp failed"; return 0; }
    if ! git clone --quiet --depth 1 --branch "$branch" "$remote" "$work" 2>/dev/null; then
      if ! git init --quiet --initial-branch "$branch" "$work" 2>/dev/null; then
        warn "git init failed"
        rm -rf "$work"
        continue
      fi
      if ! git -C "$work" remote add origin "$remote" 2>/dev/null; then
        warn "git remote add failed"
        rm -rf "$work"
        continue
      fi
    fi
    mkdir -p "$work/ci-data"
    cp -R "$staging"/. "$work/ci-data/" || { warn "copy to work dir failed"; return 0; }
    "$bin" finalize --data "$work/ci-data" --retention-days "$retention" \
      || { warn "finalize failed"; return 0; }

    if [ "$viewer" = "true" ]; then
      deploy_viewer "$work" || warn "viewer build failed (data still published)"
    fi

    git -C "$work" add -A ci-data ci 2>/dev/null || git -C "$work" add -A ci-data
    if ! git -C "$work" -c user.name=testlake -c user.email=testlake@users.noreply.github.com \
        commit --quiet -m "testlake: run ${GITHUB_RUN_ID:-local}"; then
      return 0 # 変更なし
    fi
    if git -C "$work" push --quiet origin "$branch" 2>/dev/null; then
      echo "testlake: published to $branch (attempt $attempt)"
      return 0
    fi
    warn "push rejected, retrying ($attempt/5)"
    rm -rf "$work"
  done
  warn "publish failed after 5 attempts; staging preserved at $staging"
  echo "publish_failed=true" >> "${GITHUB_OUTPUT:-/dev/null}"
  echo "staging_dir=$staging" >> "${GITHUB_OUTPUT:-/dev/null}"
  return 0
}

deploy_viewer() {
  local work="$1"
  ( cd "$GITHUB_ACTION_PATH/viewer" && npm ci --silent && npm run build --silent ) || return 1
  rm -rf "$work/ci"
  cp -R "$GITHUB_ACTION_PATH/viewer/dist" "$work/ci"
}

main
exit 0
