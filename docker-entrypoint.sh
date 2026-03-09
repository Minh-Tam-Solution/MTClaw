#!/bin/sh
set -e

case "${1:-serve}" in
  serve)
    # Managed mode: auto-upgrade (schema migrations + data hooks) before starting.
    if [ "$MTCLAW_MODE" = "managed" ] && [ -n "$MTCLAW_POSTGRES_DSN" ]; then
      echo "Managed mode: running upgrade..."
      /app/mtclaw upgrade || \
        echo "Upgrade warning (may already be up-to-date)"
    fi
    exec /app/mtclaw
    ;;
  upgrade)
    shift
    exec /app/mtclaw upgrade "$@"
    ;;
  migrate)
    shift
    exec /app/mtclaw migrate "$@"
    ;;
  onboard)
    exec /app/mtclaw onboard
    ;;
  version)
    exec /app/mtclaw version
    ;;
  *)
    exec /app/mtclaw "$@"
    ;;
esac
