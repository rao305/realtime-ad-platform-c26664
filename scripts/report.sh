#!/usr/bin/env bash
set -euo pipefail

docker compose exec -T postgres \
  psql -U ads -d ads -f /dev/stdin < platform/analytics/local_report.sql
