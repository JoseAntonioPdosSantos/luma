#!/bin/sh
# Entry point of the "backup" container: keeps a backup per day.
#
# It wakes up every BACKUP_CHECK_SECONDS and takes a backup when
#   - no backup exists yet (first start), or
#   - it is past BACKUP_HOUR (local time, see TZ) and none was taken today.
# Checking "is one due?" instead of sleeping until 03:00 means a PC that was
# off at that hour still gets its backup as soon as it is on again.
set -eu

DEST="${BACKUP_DIR:-/backups}"
HOUR="${BACKUP_HOUR:-3}"
CHECK="${BACKUP_CHECK_SECONDS:-300}"
RETRY_AFTER_FAILURE="${BACKUP_RETRY_SECONDS:-3600}"

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') scheduler: $*"; }

latest_backup() { ls -1t "$DEST"/flashcard-*.archive.gz 2>/dev/null | head -n 1 || true; }

backup_due() {
  latest=$(latest_backup)
  [ -z "$latest" ] && return 0
  [ "$(date +%-H)" -ge "$HOUR" ] || return 1
  [ "$(date -r "$latest" +%F)" != "$(date +%F)" ]
}

# Lets "docker stop" end the container promptly instead of waiting for the
# current sleep to finish.
trap 'log "stopping"; exit 0' TERM INT

mkdir -p "$DEST"
log "started: one backup per day after ${HOUR}h (timezone ${TZ:-UTC}), keeping ${BACKUP_KEEP:-1}"

while true; do
  wait_for="$CHECK"
  if backup_due; then
    if ! sh /scripts/backup.sh; then
      log "backup FAILED, trying again in ${RETRY_AFTER_FAILURE}s"
      wait_for="$RETRY_AFTER_FAILURE"
    fi
  fi
  sleep "$wait_for" &
  wait $!
done
