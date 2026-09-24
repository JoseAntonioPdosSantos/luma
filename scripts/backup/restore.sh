#!/bin/sh
# Restores a backup made by backup.sh.
#
#   restore.sh FILE               safe check: restores into a scratch database,
#                                 prints what it contains and drops it again.
#   restore.sh FILE --into-live   REPLACES the live database with the backup
#                                 (asks for confirmation first, and takes a
#                                 safety backup of the current data).
#
# FILE is a path, or just a file name from the backups folder.
set -eu

DEST="${BACKUP_DIR:-/backups}"
URI="${MONGODB_URI:-mongodb://mongo:27017}"
DB="${MONGODB_DATABASE:-flashcard}"
SCRATCH="${DB}_restore_check"

log() { echo "$(date '+%Y-%m-%d %H:%M:%S') restore: $*"; }

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
  echo "usage: restore.sh FILE [--into-live]" >&2
  exit 2
fi

file="$1"
[ -f "$file" ] || file="$DEST/$1"
[ -f "$file" ] || { echo "backup file not found: $1" >&2; exit 1; }
gzip -t "$file" || { echo "the file is not a valid gzip archive: $file" >&2; exit 1; }

print_contents() {
  mongosh "$URI/$1" --quiet --eval '
    db.getCollectionNames().sort().forEach(function (c) { print("  " + c + ": " + db[c].countDocuments()); });'
}

if [ "${2:-}" = "--into-live" ]; then
  echo "This will REPLACE the live database '$DB' with:"
  echo "  $(basename "$file")"
  echo "Stop the backend first (docker compose stop backend) so nothing writes meanwhile."
  printf "Type RESTORE to continue: "
  read -r answer
  [ "$answer" = "RESTORE" ] || { echo "cancelled"; exit 1; }

  log "taking a safety backup of the current data first"
  sh /scripts/backup.sh
  log "restoring into '$DB'"
  mongorestore --uri="$URI" --archive="$file" --gzip --drop --nsInclude="$DB.*" --quiet
  log "done; the live database now contains:"
  print_contents "$DB"
  echo "Start the backend again: docker compose start backend"
elif [ $# -eq 2 ]; then
  echo "unknown option: $2" >&2
  exit 2
else
  log "restoring into the scratch database '$SCRATCH' (the live data is untouched)"
  mongorestore --uri="$URI" --archive="$file" --gzip --drop \
    --nsInclude="$DB.*" --nsFrom="$DB.*" --nsTo="$SCRATCH.*" --quiet
  log "the backup contains:"
  print_contents "$SCRATCH"
  mongosh "$URI/$SCRATCH" --quiet --eval 'db.dropDatabase()' >/dev/null
  log "scratch database dropped; the backup is restorable"
fi
