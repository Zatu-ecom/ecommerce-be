# Restore procedure (spec 014 / T049)

Two recovery paths. Prefer the **logical dump** for point-in-time-ish recovery
of the database; use the **boot-volume backup** only to rebuild a lost host.

Backups land in `~/ecommerce-data/backups/` on VM-1, named
`ecommerce-<UTC-timestamp>.sql.gz`, newest 7 kept (`RETAIN_DAYS`).

## A. Restore the database from a logical dump

Dumps are produced by `deploy/vm1-data/backup/pg-backup.sh` via `pg_dump`.

```bash
# On VM-1
cd ~/ecommerce-data
ls -1t backups/ecommerce-*.sql.gz | head            # pick a dump
DUMP=backups/ecommerce-YYYYmmdd-HHMMSS.sql.gz

# Integrity check before trusting it
gzip -t "$DUMP"

# === DRILL: restore into a scratch database, never over the live one ===
docker exec ecommerce-postgres psql -U postgres -d postgres \
  -c 'DROP DATABASE IF EXISTS restore_drill;'
docker exec ecommerce-postgres psql -U postgres -d postgres \
  -c 'CREATE DATABASE restore_drill;'
gunzip -c "$DUMP" | docker exec -i ecommerce-postgres \
  psql -U postgres -d restore_drill -v ON_ERROR_STOP=1
# sanity: expect the same table count as production
docker exec ecommerce-postgres psql -U postgres -d restore_drill -tAc \
  "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"
docker exec ecommerce-postgres psql -U postgres -d postgres \
  -c 'DROP DATABASE restore_drill;'

# === REAL restore (only if you must roll the live DB back) ===
# 1) stop the app so it is not writing during the swap
ssh opc@130.210.26.64 'cd ~/ecommerce-app && docker compose stop app'
# 2) recreate the schema and load the dump
docker exec ecommerce-postgres psql -U postgres -d postgres \
  -c 'DROP SCHEMA public CASCADE; CREATE SCHEMA public;'
gunzip -c "$DUMP" | docker exec -i ecommerce-postgres \
  psql -U postgres -d ecommerce -v ON_ERROR_STOP=1
# 3) restart the app
ssh opc@130.210.26.64 'cd ~/ecommerce-app && docker compose start app'
```

## B. Rebuild a lost host from the boot-volume backup

1. OCI console → **Block Storage → Boot Volumes → Backups** (policy below).
2. Create a boot volume from the newest backup, attach to a fresh instance (or
   restore the existing boot volume), keep the reserved public IP.
3. Re-run `deploy/bootstrap.sh data|app` for the role, re-install the `.env`
   (secrets come from the password manager), and `docker compose up -d`.
4. Restore the DB with path A if the logical dump is newer than the boot backup.

## Drill cadence

Do path A (the `restore_drill` block) monthly. A backup that has never been
restored is a hypothesis, not a backup.
