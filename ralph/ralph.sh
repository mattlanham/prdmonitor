#!/bin/bash
set -e

MODEL="anthropic/claude-opus-4-5"
MAX_ITERATIONS=${1:-10}
SCRIPT_DIR="$(cd "$(dirname \
  "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Starting Ralph"

for i in $(seq 1 $MAX_ITERATIONS); do
  echo "═══ Iteration $i ═══"

  PROMPT=$(cat "$SCRIPT_DIR/PROMPT.md")

  OUTPUT=$(opencode run -m "$MODEL" "$PROMPT" 2>&1 \
    | tee /dev/stderr) || true

  if echo "$OUTPUT" | \
    grep -q "<promise>COMPLETE</promise>"
  then
    echo "✅ Done!"
    exit 0
  fi

  sleep 2
done

echo "⚠️ Max iterations reached"
exit 1
