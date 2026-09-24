# Backups

All of the application's data lives in one MongoDB database (`flashcard`):
users, decks, flashcards, review history **and the audio files** (GridFS).
A backup is therefore one compressed `mongodump` archive of that database.

## How it works

A `backup` service in `docker-compose.yml` runs `scripts/backup/backup-scheduler.sh`:

- It keeps **one backup per day**: the first check after `BACKUP_HOUR`
  (default 03:00, in the `BACKUP_TZ` time zone) creates it. Because it asks
  "is a backup due?" instead of sleeping until 03:00, a PC that was off at
  that hour gets its backup as soon as it is turned on.
- The very first start creates a backup immediately.
- Files are written to **`./backups/`** as
  `flashcard-YYYY-MM-DD-HHMMSS.archive.gz` (a few MB each) and belong to
  your user, so you can copy or delete them normally.
- The **last 30** are kept (`BACKUP_KEEP`); older ones are removed.
- A backup is verified (gzip integrity plus a dry-run restore) *before* it
  gets its final name, so a crashed or truncated dump never looks like a
  valid backup. If a backup fails, the scheduler retries an hour later and
  the error is in `docker compose logs backup`.

`./backups/` is git-ignored: the archives contain users' e-mail addresses
and study data.

## Settings

Set in `.env` (see `.env.example`); all are optional.

| Variable       | Default                 | Meaning |
|----------------|-------------------------|---------|
| `BACKUP_TZ`    | `America/Campo_Grande`  | Time zone for `BACKUP_HOUR` |
| `BACKUP_HOUR`  | `3`                     | Hour of day (0-23) after which the day's backup is due |
| `BACKUP_KEEP`  | `30`                    | How many backups to keep |
| `BACKUP_UID` / `BACKUP_GID` | `1000` / `1000` | Owner of the files in `./backups` (`id -u`, `id -g`) |

## Everyday commands

```sh
# Take a backup right now
docker compose exec backup sh /scripts/backup.sh

# See what exists
ls -lh backups/

# What the scheduler has been doing
docker compose logs backup
```

## Check that a backup can be restored

Do this from time to time: a backup that was never restored is only a hope.

```sh
docker compose exec backup sh /scripts/restore.sh flashcard-2026-09-20-153946.archive.gz
```

It restores into a scratch database (`flashcard_restore_check`), prints how
many documents each collection has, and drops it again. **The live data is
not touched.** Compare the numbers with your database.

## Restore for real

This **replaces** the live database with the backup.

```sh
docker compose stop backend        # nothing may write while restoring
docker compose exec backup sh /scripts/restore.sh FILE --into-live
docker compose start backend
```

The script asks you to type `RESTORE`, and first takes a safety backup of
the current data, so even a wrong choice can be undone.

### On a new machine

1. Copy the archive into `./backups/` of a fresh checkout.
2. `docker compose up -d mongo backup`
3. Run the "Restore for real" command above (the database may not exist yet).
4. `docker compose up -d`

Note that the `backup` service takes a first backup of the (empty) database
when it starts on an empty `./backups/`; that is harmless, and the restore
takes its own safety backup anyway.

## What a backup does not cover

- **Copies on the same disk do not protect against losing the disk.**
  `./backups/` is on the same disk as the Docker volume. Periodically copy it
  somewhere else (an external drive, another machine, a cloud folder), e.g.
  `rsync -a backups/ /path/to/external-drive/flashcard-backups/`.
- The `.env` file (secrets and settings) is not part of the database and is
  not in git; keep your own copy.
- Only the `flashcard` database is dumped.
