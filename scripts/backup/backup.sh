#!/bin/sh
# Creates one compressed, verified backup of the MongoDB database (cards,
# decks, users, review history and the GridFS audio files all live in it)
# and removes the oldest backups beyond BACKUP_KEEP.
#
# Runs inside the "backup" container of docker-compose.yml:
#   docker compose exec backup sh /scripts/backup.sh
set -eu

DEST="${BACKUP_DIR:-/backups}"
KEEP="${BACKUP_KEEP:-1}"
URI="${MONGODB_URI:-mongodb://mongo:27017}"
DB="${MONGODB_DATABASE:-flashcard}"
# mongodump never gives up on an unreachable server by itself (and ignores
# SIGTERM while it retries), so without a hard limit one failed run would
# block the scheduler forever. "timeout -k" sends SIGKILL after a grace period.
LIMIT="${BACKUP_TIMEOUT_SECONDS:-900}"

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') backup: $*"; }

case "$KEEP" in
  ''|*[!0-9]*|0) log "BACKUP_KEEP must be a positive integer (got '$KEEP')"; exit 2 ;;
esac

mkdir -p "$DEST"
stamp=$(date +%Y-%m-%d-%H%M%S)
partial="$DEST/.flashcard-$stamp.partial"
final="$DEST/flashcard-$stamp.archive.gz"
# A half-written file must never look like a backup: it is only renamed to
# its final name after it has been verified.
trap 'rm -f "$partial"' EXIT

log "dumping database '$DB'"
timeout -k 10 "$LIMIT" mongodump --uri="$URI" --db="$DB" --archive="$partial" --gzip --quiet

[ -s "$partial" ] || { log "the dump is empty"; exit 1; }
gzip -t "$partial"
# A dry-run restore reads the whole archive and fails if it is damaged.
timeout -k 10 "$LIMIT" mongorestore --uri="$URI" --archive="$partial" --gzip --dryRun --quiet

mv "$partial" "$final"
log "created $(basename "$final") ($(du -h "$final" | cut -f1))"

# Newest first; everything past the first KEEP files is removed.
count=0
for f in $(ls -1t "$DEST"/flashcard-*.archive.gz 2>/dev/null); do
  count=$((count + 1))
  if [ "$count" -gt "$KEEP" ]; then
    rm -f -- "$f"
    log "removed old backup $(basename "$f")"
  fi
done
