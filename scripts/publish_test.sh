#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
ROOT="$PWD"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

export GITHUB_ACTION_PATH="$ROOT"
export RUNNER_TEMP="$TMP/runner"
mkdir -p "$RUNNER_TEMP"
export INPUT_BRANCH=gh-pages INPUT_VIEWER=false INPUT_RETENTION_DAYS=400
export GITHUB_TOKEN=dummy GITHUB_REPOSITORY=local/test GITHUB_RUN_ID=1
export GITHUB_RUN_ATTEMPT=1 GITHUB_JOB=test GITHUB_WORKFLOW=CI
export GITHUB_REF_NAME=main GITHUB_SHA=cafebabe GITHUB_EVENT_NAME=push RUNNER_OS=Linux
export TESTLAKE_NOW=2026-07-08T10:00:00Z
export GITHUB_OUTPUT="$TMP/out"

git init --quiet --bare "$TMP/origin.git"
export TESTLAKE_REMOTE_URL="$TMP/origin.git"
export INPUT_REPORTS="$ROOT/collector/internal/junit/testdata/pytest.xml"

# 1) 初回: ブランチが存在しない状態から publish できる
bash scripts/publish.sh
git clone --quiet --branch gh-pages "$TMP/origin.git" "$TMP/check1"
test -f "$TMP/check1/ci-data/manifest.json" || { echo "FAIL: no manifest"; exit 1; }
ls "$TMP/check1/ci-data/tests/date=2026-07-08/" | grep -q '1-test-1.parquet' || { echo "FAIL: no parquet"; exit 1; }

# 2) 2回目: 既存ブランチへの追記
export GITHUB_RUN_ID=2
bash scripts/publish.sh
git clone --quiet --branch gh-pages "$TMP/origin.git" "$TMP/check2"
ls "$TMP/check2/ci-data/tests/date=2026-07-08/" | grep -q '2-test-1.parquet' || { echo "FAIL: second run missing"; exit 1; }
ls "$TMP/check2/ci-data/tests/date=2026-07-08/" | grep -q '1-test-1.parquet' || { echo "FAIL: first run lost"; exit 1; }

# 3) 壊れた入力でも exit 0 かつ ::warning:: を出すこと(never-fail invariant)
#    - INPUT_REPORTS がどこにも一致しない glob
#    - TESTLAKE_REMOTE_URL が存在しないパス(push が5回とも失敗する)
export GITHUB_RUN_ID=3
export INPUT_REPORTS="$TMP/nowhere/does-not-exist/*.xml"
export TESTLAKE_REMOTE_URL="$TMP/no-such-remote.git"
set +e
bash scripts/publish.sh >"$TMP/scenario3.log" 2>&1
rc=$?
set -e
cat "$TMP/scenario3.log"
test "$rc" -eq 0 || { echo "FAIL: publish.sh exited $rc (expected 0) for broken input"; exit 1; }
grep -q '::warning::testlake:' "$TMP/scenario3.log" || { echo "FAIL: no ::warning:: emitted for broken input"; exit 1; }

# 4) go build failure must also exit 0 かつ ::warning:: を出すこと
#    - GITHUB_ACTION_PATH を empty dir に指す (collector/ subdir がない)
#    - ( cd "$GITHUB_ACTION_PATH/collector" && go build ... ) が失敗する
export GITHUB_RUN_ID=4
export INPUT_REPORTS="$ROOT/collector/internal/junit/testdata/pytest.xml"
export TESTLAKE_REMOTE_URL="$TMP/origin.git"
empty_dir="$TMP/empty"
mkdir -p "$empty_dir"
set +e
GITHUB_ACTION_PATH="$empty_dir" bash scripts/publish.sh >"$TMP/scenario4.log" 2>&1
rc=$?
set -e
cat "$TMP/scenario4.log"
test "$rc" -eq 0 || { echo "FAIL: publish.sh exited $rc (expected 0) for go build failure"; exit 1; }
grep -q '::warning::testlake:' "$TMP/scenario4.log" || { echo "FAIL: no ::warning:: emitted for go build failure"; exit 1; }

echo "publish_test: OK"
