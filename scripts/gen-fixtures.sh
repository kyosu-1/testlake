#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
OUT="viewer/tests/fixtures/ci-data"
rm -rf "$OUT" && mkdir -p "$OUT"
BIN="$(mktemp -d)/testlake"
( cd collector && go build -o "$BIN" ./cmd/testlake )
TMP="$(mktemp -d)"

junit() { # junit <path> <TestLogin-outcome> <TestPay-ms> <extra-xml>
  cat > "$1" <<EOF
<?xml version="1.0"?>
<testsuite name="suite" tests="3">
  <testcase classname="auth" name="TestLogin" time="0.05">$2</testcase>
  <testcase classname="billing" name="TestPay" time="$3"/>
  $4
</testsuite>
EOF
}

collect() { # collect <run_id> <attempt> <sha> <now> <report>
  GITHUB_RUN_ID="$1" GITHUB_RUN_ATTEMPT="$2" GITHUB_SHA="$3" \
  GITHUB_WORKFLOW=CI GITHUB_JOB=test GITHUB_REF_NAME=main \
  GITHUB_EVENT_NAME=push RUNNER_OS=Linux \
  TESTLAKE_JOB_STATUS=success TESTLAKE_NOW="$4" \
  "$BIN" collect --reports "$5" --data "$OUT"
}

junit "$TMP/r1.xml" '<failure message="boom">t</failure>' 0.100 ''
collect 1 1 aaa 2026-06-25T10:00:00Z "$TMP/r1.xml"
junit "$TMP/r2.xml" '' 0.100 ''
collect 1 2 aaa 2026-06-25T10:10:00Z "$TMP/r2.xml"
junit "$TMP/r3.xml" '' 0.300 ''
collect 3 1 bbb 2026-07-06T10:00:00Z "$TMP/r3.xml"
junit "$TMP/r4.xml" '' 0.320 '<testcase classname="shop" name="TestCheckout" time="0.01"><failure message="broken">t</failure></testcase>'
collect 4 1 ccc 2026-07-07T10:00:00Z "$TMP/r4.xml"

"$BIN" finalize --data "$OUT" --retention-days 400
echo "fixtures written to $OUT"
