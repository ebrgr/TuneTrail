---
title: Installation
description: Guide on how to install Koito to start tracking your listening history.
---

## Docker

By far the easiest way to get up and running with Koito is using docker. Here is an example Docker Compose file to get you up and running in minutes:

```yaml title="compose.yaml"
services:
  koito:
    image: ghcr.io/ebrgr/tunetrail:latest
    container_name: koito
    ports:
      - "4110:4110"
    volumes:
      - ./tunetrail:/etc/tunetrail
    restart: unless-stopped
```

:::note
Koito uses SQLite for all new installations as of `v0.2.1`. If you are running a version older than that, you will need to set `KOITO_SQLITE_ENABLED=true`.
:::

You can find a full list of configuration options in the [configuration reference](/reference/configuration).

## Build from source

If you don't want to use docker, you can also build the application from source.

**Note that no support is provided when running Koito built from source.**

Install Go at the version declared in `go.mod` and `make`. The Go application
builds with `CGO_ENABLED=0` and does not require libvips or a C compiler. Node.js
is only needed for interface script tests and documentation; the optional Go race
detector needs a C compiler.

Then, clone the repository and execute the build command using the included Makefile:

```sh
git clone https://github.com/ebrgr/TuneTrail.git && cd TuneTrail
make build
```

When the build is finished, you can run the executable at the root of the directory.

```sh
./tunetrail
```

Then, navigate your browser to `localhost:4110` to enter your Koito instance.

## Listening profiles and upgrades

The directory at `/` lists profiles. Sign in at `/login` to manage your
account and profile. Administrators can manage all accounts and profiles.

For profile settings and player keys, see [Listening profiles](/guides/profiles).
For upgrading an existing Koito SQLite installation or restoring from backup,
follow [Multiuser operations](/guides/multiuser-operations).
