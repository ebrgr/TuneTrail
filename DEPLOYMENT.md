# Deploying TuneTrail

TuneTrail adds public listening profiles and administrator-only management to
Koito. The current source version is
**TuneTrail 0.9**, a prerelease intended for testing with a backup of your data.

The target image is `ghcr.io/ebrgr/tunetrail:0.9`, built for Linux amd64
and arm64. Publish the matching image before pulling this tag, or build it locally. The image serves both the API and compiled interface on port 4110;
Node.js, Go, and Vite are not required on the deployment host.

## Start a fresh installation

Install Docker Engine/Desktop with the Docker Compose plugin, then clone this
repository or extract the release's deployment bundle. From its root:

```sh
cp deploy/.env.example deploy/.env
```

Edit `deploy/.env`. Its `KOITO_*` names are read by Docker Compose and stay as
they are until 1.0; inside the container TuneTrail uses `TUNETRAIL_*` variables
(see the configuration reference). Set `KOITO_ADMIN_PASSWORD` to a unique password of at least
eight characters. `KOITO_ADMIN_USERNAME` defaults to `admin`. These settings
create the administrator only when the database has no users.

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml pull
docker compose --env-file deploy/.env -f deploy/compose.yaml up -d
docker compose --env-file deploy/.env -f deploy/compose.yaml ps
```

Open <http://localhost:4110>. Choose a public profile, or use
**Settings → Administration** to sign in and create/edit profiles and API keys.
The initial data directory is `deploy/data` and is mounted at `/etc/koito`.
Keep `deploy/.env` and the data directory outside version control.

The published host port binds to `127.0.0.1` by default. For remote access, use
your reverse proxy with HTTPS or explicitly set `KOITO_HOST_BIND` to the host
address you intend to expose. The container's internal API binds to all container
interfaces, independently of this host setting.

## Upgrade an existing SQLite v2 installation

**Stop the old Koito process/container and back up its complete data directory
before testing this release.** Do not run two versions against the same database.
Preserve the image cache and all files alongside the legacy `koito.db`, including any WAL files. Stop every old instance before migration; a backup snapshot cannot prevent subsequent writes to the old database.

Set `KOITO_DATA_DIR` in `deploy/.env` to the absolute path of that existing data
directory, rather than a new empty directory. Set the intended host port, and run
the `pull` and `up -d` commands above. Keep the existing administrator's login:
changing bootstrap environment variables does not reset an existing password.

On first startup, the application:

1. Creates a consistent SQLite backup, including committed WAL, named
   `koito.db.pre-tunetrail-*.bak` in the data directory (permissions 0600).
2. Creates `tunetrail.db` from that backup and, for v2, applies the single
   `003_listening_profiles.sql` migration to reach v3.
3. Preserves administrator credentials, existing tokens, listens, and catalog.
4. Assigns a single legacy user's history/keys to `/default`; multiple legacy
   users receive separate profiles.

The legacy file and backup are retained. If both databases exist, the server
checks migration provenance; a changed legacy file, including an external WAL
checkpoint, causes a conservative refusal. Stop all instances and reconcile
the files manually rather than deleting or overwriting them. New installations
create `tunetrail.db` directly.

The migration includes profiles, associations, indexes, per-profile relays, and
protection against transferring a key. If backup or migration fails, startup
fails; review logs before retrying. Old experimental SUM databases that already
used the earlier v3/v4 layout need a v2 backup restored before this consolidated
migration. Do not relabel their entries in `goose_db_version`.

Automatic directory import is disabled in the supplied Compose file. Import
history through the Administration tab with an explicit destination profile.
Profiles are public; only administrative operations require a password. The
volume contains credentials and listening data and must be retained across
container upgrades.

## Recover an interrupted or refused migration

The migration publishes `tunetrail.db` in a single atomic step, after the
backup, the copy, the schema upgrade and the checks have finished. An
interrupted start (crash, `kill -9`, power loss or a full disk) therefore
leaves either no `tunetrail.db` or a complete one; `koito.db` is never
modified. Always stop every TuneTrail and Koito container before acting.

1. **No `tunetrail.db` in the data directory.** The migration did not finish.
   Free disk space if the log reports it, then start the container again; it
   repeats the migration from `koito.db` and creates a new backup.
2. **`tunetrail.db` exists.** The migration finished. Start normally.
3. **Leftover files.** An interrupted start can leave `.tunetrail-migrate-*.db`
   files and extra `koito.db.pre-tunetrail-*.bak` backups. They are never
   used automatically. After a successful start, find the backup in use with
   `sqlite3 tunetrail.db "SELECT backup_name FROM tunetrail_migration"`, keep
   it, and remove the `.tunetrail-migrate-*.db` files. Keep the other backups
   until you no longer need them.
4. **"orphaned database sidecar".** A `tunetrail.db-wal`, `-shm` or `-journal`
   exists without `tunetrail.db`. Move those files out of the data directory
   (do not delete them yet) and start again.
5. **"legacy database changed after migration" or "conflicting databases".**
   `koito.db` was written after the migration, usually because an older
   version ran against the same directory. Choose one copy:
   - keep the newer history in `tunetrail.db`: move `koito.db` (and its
     `-wal`/`-shm` files) out of the data directory, then start; or
   - start over from the legacy database: move `tunetrail.db` and its
     `-wal`/`-shm` files out of the data directory, then start; listens
     recorded only in `tunetrail.db` are lost unless you export them first.

To restore from the backup, stop the container, create a new data directory,
copy the backup in use into it as `koito.db` (permissions `0600`), copy the
image cache, point `KOITO_DATA_DIR` at the new directory and start. The
migration runs again from that copy.

## Logs, health, and shutdown

```sh
docker compose --env-file deploy/.env -f deploy/compose.yaml logs --tail=100 koito
curl --fail http://127.0.0.1:4110/apis/web/v1/health
docker compose --env-file deploy/.env -f deploy/compose.yaml stop
```

The image includes a healthcheck. Compose gives the application up to 30 seconds
to shut down gracefully. `stop` and `down` preserve the host data directory.

## Rollback

The migration intentionally does not offer a destructive SQL downgrade.
Versions up to 0.5 read `koito.db`; later versions read `tunetrail.db`. A
rollback to such a version shows the history as it was before the upgrade:
listens, keys and settings recorded afterwards exist only in `tunetrail.db`.
Export the affected profiles from the Administration tab first if you need
them, and import them again after upgrading.

1. Stop the container and back up the complete data directory.
2. Either start the previous version against the same directory (it reads the
   untouched `koito.db`), or restore the backup in use as `koito.db` in a new
   directory, as described above, and start the previous version there.
3. The previous version upgrades `koito.db` in place. Upgrading again later
   in the same directory is refused with "legacy database changed after
   migration"; follow step 5 of the recovery procedure, or upgrade from a new
   directory restored from the backup.

These paths were exercised with TuneTrail 0.5 and 0.9 on fictional data
(`scripts/validate_rollback.py`).

## Container registry access

Public GHCR packages can be pulled without logging in. A package initially
created as private requires a GitHub login with package-read access until the
owner makes it public. See [GitHub's Container registry documentation](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry).

Version tags identify releases; `sha-...` tags identify their source commit.
Each image publication also updates `ghcr.io/ebrgr/tunetrail:latest`, including
prerelease versions and manual workflow runs. Set `KOITO_IMAGE_TAG=latest` in
`deploy/.env` to follow it, or choose a version tag for a fixed release. Pulling
the image and recreating the container are still necessary to apply updates.

## Build locally

```sh
docker build -t tunetrail:local .
```

.dockerignore excludes local databases, credentials, caches, and development
dependencies. The application is built with Go 1.25 and CGO disabled; it does
not require libvips at build or runtime. The runtime retains CA certificates,
timezone data and curl for the healthcheck.

For local development and validation, see [DEVELOPMENT.md](DEVELOPMENT.md).

Current source: **TuneTrail 0.9**. A matching image must be published before pulling this tag. For an unpublished version, use the local build instructions above. Version numbers advance once per accepted task published with `make issue.publish` (0.9 becomes 0.10); pushing its `v0.x` tag builds and publishes the matching image; 1.0 is reserved for a later decision. Historical SUM tags remain unchanged.
