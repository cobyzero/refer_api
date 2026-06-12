#!/usr/bin/env bash
set -u

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPORT_DIR="$PROJECT_DIR/test-reports"
TIMESTAMP="$(date '+%Y-%m-%d_%H-%M-%S')"
HTML_REPORT="$REPORT_DIR/test-report-$TIMESTAMP.html"
RAW_LOG="$REPORT_DIR/test-run-$TIMESTAMP.log"

mkdir -p "$REPORT_DIR"

escape_html() {
  sed -e 's/&/\&amp;/g' \
      -e 's/</\&lt;/g' \
      -e 's/>/\&gt;/g' \
      -e 's/"/\&quot;/g' \
      -e "s/'/\&#39;/g"
}

run_test_block() {
  local label="$1"
  local cmd="$2"
  local output_file="$3"
  local status_file="$4"
  local duration_file="$5"

  local start end duration status
  start="$(date +%s)"
  (
    cd "$PROJECT_DIR" || exit 1
    bash -lc "$cmd"
  ) >"$output_file" 2>&1
  status=$?
  end="$(date +%s)"
  duration=$((end - start))

  echo "$status" > "$status_file"
  echo "$duration" > "$duration_file"

  {
    echo "[$label] command: $cmd"
    echo "[$label] exit code: $status"
    echo "[$label] duration(s): $duration"
    echo "[$label] output:"
    cat "$output_file"
    echo ""
    echo "------------------------------------------------------------"
    echo ""
  } >> "$RAW_LOG"
}

UNIT_OUT="$REPORT_DIR/unit-$TIMESTAMP.txt"
E2E_OUT="$REPORT_DIR/e2e-$TIMESTAMP.txt"
COV_OUT="$REPORT_DIR/cov-$TIMESTAMP.txt"

UNIT_STATUS_FILE="$REPORT_DIR/unit-status-$TIMESTAMP.txt"
E2E_STATUS_FILE="$REPORT_DIR/e2e-status-$TIMESTAMP.txt"
COV_STATUS_FILE="$REPORT_DIR/cov-status-$TIMESTAMP.txt"

UNIT_DURATION_FILE="$REPORT_DIR/unit-duration-$TIMESTAMP.txt"
E2E_DURATION_FILE="$REPORT_DIR/e2e-duration-$TIMESTAMP.txt"
COV_DURATION_FILE="$REPORT_DIR/cov-duration-$TIMESTAMP.txt"

echo "Test execution started at $(date)" > "$RAW_LOG"
echo "" >> "$RAW_LOG"

run_test_block "Unit Tests" "npm run test -- --watchman=false" "$UNIT_OUT" "$UNIT_STATUS_FILE" "$UNIT_DURATION_FILE"
run_test_block "E2E Tests" "npm run test:e2e" "$E2E_OUT" "$E2E_STATUS_FILE" "$E2E_DURATION_FILE"
run_test_block "Coverage Tests" "npm run test:cov -- --watchman=false" "$COV_OUT" "$COV_STATUS_FILE" "$COV_DURATION_FILE"

UNIT_STATUS="$(cat "$UNIT_STATUS_FILE")"
E2E_STATUS="$(cat "$E2E_STATUS_FILE")"
COV_STATUS="$(cat "$COV_STATUS_FILE")"

UNIT_DURATION="$(cat "$UNIT_DURATION_FILE")"
E2E_DURATION="$(cat "$E2E_DURATION_FILE")"
COV_DURATION="$(cat "$COV_DURATION_FILE")"

if [[ "$UNIT_STATUS" -eq 0 && "$E2E_STATUS" -eq 0 && "$COV_STATUS" -eq 0 ]]; then
  OVERALL_STATUS="SUCCESS"
  OVERALL_CLASS="ok"
  EXIT_CODE=0
else
  OVERALL_STATUS="FAILED"
  OVERALL_CLASS="fail"
  EXIT_CODE=1
fi

to_badge() {
  local status="$1"
  if [[ "$status" -eq 0 ]]; then
    echo "PASS"
  else
    echo "FAIL"
  fi
}

UNIT_BADGE="$(to_badge "$UNIT_STATUS")"
E2E_BADGE="$(to_badge "$E2E_STATUS")"
COV_BADGE="$(to_badge "$COV_STATUS")"

UNIT_CLASS="fail"; [[ "$UNIT_STATUS" -eq 0 ]] && UNIT_CLASS="ok"
E2E_CLASS="fail"; [[ "$E2E_STATUS" -eq 0 ]] && E2E_CLASS="ok"
COV_CLASS="fail"; [[ "$COV_STATUS" -eq 0 ]] && COV_CLASS="ok"

UNIT_LOG_HTML="$(escape_html < "$UNIT_OUT")"
E2E_LOG_HTML="$(escape_html < "$E2E_OUT")"
COV_LOG_HTML="$(escape_html < "$COV_OUT")"

cat > "$HTML_REPORT" <<EOF
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Test Report - $TIMESTAMP</title>
  <style>
    :root {
      --bg: #f6f8fc;
      --card: #ffffff;
      --ink: #12223a;
      --muted: #5c6b80;
      --ok: #1f8a4c;
      --ok-bg: #e9f8ef;
      --fail: #b42318;
      --fail-bg: #feefee;
      --border: #dbe3ee;
      --accent: #0b6bcb;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: "Segoe UI", Tahoma, Arial, sans-serif;
      color: var(--ink);
      background: linear-gradient(120deg, #eef3fb 0%, #f8fafc 60%, #f1f7ff 100%);
    }
    .container {
      max-width: 1080px;
      margin: 32px auto;
      padding: 0 16px 32px;
    }
    .hero {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 14px;
      padding: 20px;
      box-shadow: 0 8px 24px rgba(16, 39, 71, 0.07);
    }
    h1 { margin: 0 0 8px; font-size: 26px; }
    .subtitle { color: var(--muted); margin: 0 0 14px; }
    .status {
      display: inline-block;
      padding: 8px 12px;
      border-radius: 999px;
      font-weight: 700;
      letter-spacing: 0.3px;
    }
    .status.ok { background: var(--ok-bg); color: var(--ok); }
    .status.fail { background: var(--fail-bg); color: var(--fail); }
    .grid {
      margin-top: 18px;
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 12px;
    }
    .tile {
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 14px;
    }
    .tile h3 { margin: 0 0 6px; font-size: 15px; }
    .badge {
      display: inline-block;
      font-size: 12px;
      font-weight: 700;
      padding: 4px 10px;
      border-radius: 999px;
      margin-bottom: 8px;
    }
    .badge.ok { background: var(--ok-bg); color: var(--ok); }
    .badge.fail { background: var(--fail-bg); color: var(--fail); }
    .meta { color: var(--muted); font-size: 13px; }
    details {
      margin-top: 16px;
      background: var(--card);
      border: 1px solid var(--border);
      border-radius: 12px;
      overflow: hidden;
    }
    summary {
      cursor: pointer;
      list-style: none;
      padding: 14px 16px;
      font-weight: 600;
      border-bottom: 1px solid var(--border);
    }
    summary::-webkit-details-marker { display: none; }
    pre {
      margin: 0;
      padding: 16px;
      overflow: auto;
      background: #0f1c2e;
      color: #e8eef8;
      font-size: 12px;
      line-height: 1.4;
    }
    .footer {
      margin-top: 16px;
      color: var(--muted);
      font-size: 12px;
    }
    a { color: var(--accent); text-decoration: none; }
  </style>
</head>
<body>
  <div class="container">
    <div class="hero">
      <h1>API Test Execution Report</h1>
      <p class="subtitle">Project: $PROJECT_DIR</p>
      <span class="status $OVERALL_CLASS">$OVERALL_STATUS</span>
      <div class="grid">
        <div class="tile">
          <h3>Unit Tests</h3>
          <span class="badge $UNIT_CLASS">$UNIT_BADGE</span>
          <div class="meta">Exit code: $UNIT_STATUS</div>
          <div class="meta">Duration: ${UNIT_DURATION}s</div>
          <div class="meta">Command: <code>npm run test -- --watchman=false</code></div>
        </div>
        <div class="tile">
          <h3>E2E Tests</h3>
          <span class="badge $E2E_CLASS">$E2E_BADGE</span>
          <div class="meta">Exit code: $E2E_STATUS</div>
          <div class="meta">Duration: ${E2E_DURATION}s</div>
          <div class="meta">Command: <code>npm run test:e2e</code></div>
        </div>
        <div class="tile">
          <h3>Coverage</h3>
          <span class="badge $COV_CLASS">$COV_BADGE</span>
          <div class="meta">Exit code: $COV_STATUS</div>
          <div class="meta">Duration: ${COV_DURATION}s</div>
          <div class="meta">Command: <code>npm run test:cov -- --watchman=false</code></div>
        </div>
      </div>
    </div>

    <details>
      <summary>Unit Tests Log</summary>
      <pre>$UNIT_LOG_HTML</pre>
    </details>

    <details>
      <summary>E2E Tests Log</summary>
      <pre>$E2E_LOG_HTML</pre>
    </details>

    <details>
      <summary>Coverage Log</summary>
      <pre>$COV_LOG_HTML</pre>
    </details>

    <p class="footer">
      Generated: $(date) | Raw log: $RAW_LOG
    </p>
  </div>
</body>
</html>
EOF

echo "HTML report generated at:"
echo "$HTML_REPORT"
echo ""
echo "Raw log:"
echo "$RAW_LOG"

exit "$EXIT_CODE"

