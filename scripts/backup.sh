#!/usr/bin/env bash
# Nightly pg_dump -> object storage (Cloudflare R2 or Backblaze B2),
# with a retention window. Cron-schedule this, e.g.:
#   0 3 * * * /opt/family-app-suite/scripts/backup.sh >> /var/log/family-app-suite/backup.log 2>&1
#
# Requires `rclone` configured with a remote named by RCLONE_REMOTE
# (e.g. `rclone config` once, pointed at R2/B2), and DATABASE_URL set.
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"
: "${RCLONE_REMOTE:?RCLONE_REMOTE is required, e.g. r2:family-app-suite-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"

timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
dump_file="/tmp/family-app-suite-${timestamp}.sql.gz"

pg_dump "${DATABASE_URL}" | gzip > "${dump_file}"
rclone copy "${dump_file}" "${RCLONE_REMOTE}/"
rm -f "${dump_file}"

# Prune backups older than the retention window.
rclone delete --min-age "${RETENTION_DAYS}d" "${RCLONE_REMOTE}/"

echo "backup complete: ${dump_file##*/}"
