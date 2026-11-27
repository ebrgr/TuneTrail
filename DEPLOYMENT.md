# Deploying TuneTrail

TuneTrail adds public listening profiles and administrator-only management to
Koito. The current source version is
**TuneTrail 0.79**, a prerelease intended for testing with a backup of your data.

The target image is `ghcr.io/ebrgr/tunetrail:0.79`, built for Linux amd64
and arm64. Publish the matching image before pulling this tag, or build it locally. The image serves both the API and compiled interface on port 4110;
Node.js, Go, and Vite are not required on the deployment host.

## Start a fresh installation

Install Docker Engine/Desktop with the Docker Compose plugin, then clone this
repository or extract the release's deployment bundle. From its root:

```sh
cp deploy/.env.example deploy/.env
```

Edit `deploy/.env`. Its `KOITO_*` names are read by Docker Compose and stay as
they are until 2.0; inside the container TuneTrail uses `TUNETRAIL_*` variables
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
The initial data directory is `deploy/data` and is mounted at `/etc/tunetrail`.
Keep `deploy/.env` and the data directory outside version control.

The published host port binds to `127.0.0.1` by default. For remote access, use
your reverse proxy with HTTPS or explicitly set `KOITO_HOST_BIND` to the host
address you intend to expose. The container's internal API binds to all container
interfaces, independently of this host setting.

## Upgrade from the `koito-sum` compose project

Since 0.79 the compose project and service are named `tunetrail`, the data is
mounted at `/etc/tunetrail` and variables use the `TUNETRAIL_` prefix. Keep your
`deploy/.env`: its `KOITO_*` names are unchanged, so it still points at the same
host data directory.

1. With the **old** `deploy/compose.yaml` still in place, stop the old project:
   `docker compose --env-file deploy/.env -f deploy/compose.yaml down` (or, if the
   file was already replaced, `docker compose -p koito-sum down`). Do not use `-v`.
2. Replace `deploy/compose.yaml` with the new one.
3. Start: `docker compose --env-file deploy/.env -f deploy/compose.yaml up -d --remove-orphans`.
4. Check that only the `tunetrail` container runs: `docker ps`.

If the old container is still running on the same data directory, the new one
refuses to start ("another TuneTrail server is already using this data
directory"). This guard relies on file locks: it works on Linux hosts (including
Unraid) and Docker volumes, but bind mounts from macOS or Windows through
Docker Desktop do not share locks between containers, so always stop the old
project first. Installations with their own compose file or an Unraid template
that mount `/etc/koito` keep working without changes until 2.0; the log asks to
move the mount to `/etc/tunetrail`. The administrator signs in again once,
because the session cookie was renamed.

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
2. Migrates a copy of the backup from schema 2 through the current schema (23) before
   publishing it as `tunetrail.db`.
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
Profiles start public and the owner or an administrator can protect their
listening history. Keep the volume across container upgrades. See
[Multiuser operations](docs/src/content/docs/guides/multiuser-operations.md)
for account activation, local visibility, and backup restoration.

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
docker compose --env-file deploy/.env -f deploy/compose.yaml logs --tail=100 tunetrail
curl --fail http://127.0.0.1:4110/apis/web/v1/health
docker compose --env-file deploy/.env -f deploy/compose.yaml stop
```

The image includes a healthcheck. Compose gives the application up to 30 seconds
to shut down gracefully. `stop` and `down` preserve the host data directory.

## Historical schema v11 migration (ISSUE-134)

Stop the service in a maintenance window before upgrading from a v9 database
(TuneTrail 0.44). Keep free disk space of at least three times the
database size in the data directory and in the SQLite temporary directory
(`TMPDIR`). On first start TuneTrail writes a verified
`.pre-occurrence-ingest-*.bak` backup next to the database, then migrates it to
v11. On the test hardware a database with 300,000 receipts needed about 21
seconds before the health endpoint answered and a backup of about 260 MB.

Current releases apply this migration on the way to the current schema. Occurrence
ingestion is active for every profile since TuneTrail 0.65. Versions 0.46 and
earlier refuse a v11 database. To go back, stop the service and restore the backup; writes made
after the upgrade are lost, and relay deliveries already made stay at the
destination (ISSUE-134).

### Physical database backups

Plan for longer downtime and more free disk space as the database grows.
During the upgrade the live database, WAL, and physical backup can coexist.
Budget for all three and allow time for startup maintenance before serving
requests.

Treat every physical backup or full database copy as sensitive as the live
database. It preserves the data present in the source schema, including
password hashes, API keys, and any configured relay token in plaintext.
TuneTrail creates automatic upgrade backups alongside the database with
owner-only permissions (`0600`), including `.pre-relay-delivery-*.bak`,
`.pre-occurrence-ingest-*.bak`, and `koito.db.pre-tunetrail-*.bak`.
Preserve those permissions on copies. Keep the backups out of shared
locations and do not send them to third parties. Protect them and delete
obsolete copies using the same procedures as for the database itself.
History exports and the administrative relay GET omit the token.

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

These paths were exercised with TuneTrail 0.5 and 0.79 on fictional data
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

Current source: **TuneTrail 0.79**. A matching image must be published before pulling this tag. For an unpublished version, use the local build instructions above. Version numbers advance once per accepted task published with `make issue.publish` (0.79 becomes 0.79); pushing its `v0.x` tag builds and publishes the matching image; 1.0 is reserved for a later decision. Historical SUM tags remain unchanged.

## External web assets (ISSUE-066)

Install the executable together with the `assets/` and `langs/` directories from the same release. A binary alone no longer contains the web interface. Without an override, web assets are loaded from `assets/` beside the real executable, following executable symlinks. `TUNETRAIL_WEB_DIR` selects another asset root explicitly; an invalid override fails startup. This is the asset root, not its templates subdirectory.

The server validates and loads a single immutable in-memory generation at startup. Restart after changing templates, CSS, JavaScript or palettes. There is no automatic reload or fallback to embedded files. Deploy matching binary+assets as a complete release in a trusted directory, separate from writable database/import/cache storage. Disk edits after startup do not modify the running generation. Root ownership and read-only modes are set on assets in the image; running the entire container as UID65532 remains the coordinated work of ISSUE-102.

The image already copies assets into its final stage, including non-served license files. Only allowlisted static file types are exposed, with no directory listings or source-template/license routes. Static MIME types are explicit and responses use nosniff. Modification times remain absent; static assets use content-derived ETags and versioned URLs as described below.

Language catalogs retain their separate existing lookup contract: an explicit `TUNETRAIL_LANGS_DIR` is used without fallback if invalid; otherwise catalogs beside the resolved executable are preferred, then a checkout with `go.mod` may be found from cwd for compatibility. Packaged/server installations should always use the beside-executable catalogs or an explicit root. This compatibility fallback applies to language catalogs only, never web assets.

For local distribution, use `make package OUT=/path/new-archive.tar.gz`. The archive includes `tunetrail`, `assets/`, `langs/` and `LICENSE`; icons licenses remain within the assets distribution but are not served. Sources are not embedded in the executable.

## Static asset caching and history (ISSUE-106)

Startup builds an immutable catalog from the same validated external snapshot. Static CSS, scripts, icons and palette data use SHA-256 URLs such as `/web-assets/ui.css?v=<hash>`. CSS dependencies are versioned before hashing the final stylesheet. The preview uses identical resource bytes under `/preview-assets/`. Licenses and HTML sources remain outside the public catalog.

A matching canonical hash receives `Cache-Control: public, max-age=31536000, immutable`. A fixed URL without a query receives `no-cache` and the same strong ETag. Old/invalid hashes and unexpected queries return 404 with `no-store`, never current bytes under an old immutable URL. Conditional and range requests use the final representation. Fixed legacy `/public/*` and `/favicon.svg` aliases remain revalidatable; the legacy web manifest keeps its fixed icon URLs. Preview static resources are also public and can be stored by a proxy; generated demo images and user artwork have separate policies.

**Only the running generation is retained.** Deploy binary and assets together and restart. Previous hashed URLs may become unavailable after a release change; an old tab may need a full reload, especially if its scripts or deferred resources have been evicted from the browser cache. The application does not persist historical asset bundles or promise uninterrupted old tabs or mixed-generation rolling deployments.

Public successful documents use `private, no-cache` with Cookie/Accept-Language variance; document errors, administration, login and mutation/CSRF responses remain `no-store`. Browser back/forward cache is separate from HTTP cache. On a persisted public restore the app recreates request controllers and revalidates administrative authority. A background HEAD checks locale, UI generation and server-resolved timezone/week context and may reload a changed document; it can execute normal SSR/SQL and is not an optimization of database queries. Administrative persisted restores reload to refresh authorization and forms, with sensitive content hidden during suspension. Public statistics otherwise retain their historical snapshot until refreshed.

Reverse proxies must preserve the full query and these cache directives, and must not cache private HTML as public. When a proxy adds compression it must handle representation-specific ETags and encoding variance correctly. Do not enable arbitrary compression while reusing an incompatible strong ETag. Test the actual TLS/proxy configuration, not just loopback HTTP.

The CSS catalog accepts local leaf dependencies only: use literal `url(...)` (no image-set/src functions), no preexisting query on a local asset, no `@import` or CSS-to-CSS dependency. Relative paths resolve against the published stylesheet path, not its physical source directory. Invalid dependencies fail startup. ISSUE-068 may extend this after deterministic concatenation; ISSUE-070 adds local WOFF2 entries without exposing license files.
