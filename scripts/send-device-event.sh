#!/bin/sh
set -eu

curl --fail-with-body \
  --request POST \
  --header 'Content-Type: application/json' \
  --data '{"shipment_id":"SHP-42","device_id":"DEV-7","status":"moving","temperature_c":4.5,"damaged":false,"occurred_at":"2026-09-28T10:00:00Z"}' \
  "${DASHBOARD_URL:-http://localhost:8080}/shipments/events"
