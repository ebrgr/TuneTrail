<div align="center">
  <img src="assets/icon.png" alt="TuneTrail logo" width="180">

# TuneTrail

**Your music. Your journey.**

A self-hosted listening history, based on
<img src="assets/koito-original.svg" alt="" width="18"> Koito engine version 0.3.2.
</div>

TuneTrail records your listening history and presents statistics for artists, albums and tracks. It accepts scrobbles from ListenBrainz-compatible players and lets you manage separate listening profiles.

TuneTrail simplifies Koito's architecture into a single Go application with server-rendered HTML, CSS and minimal JavaScript. The React frontend and native image-processing dependencies have been removed; the application runs without CGO or libvips. SQLite stores your data locally.

## Features

- Listening profiles, history, rankings and retrospectives.
- ListenBrainz-compatible submissions, profile-specific keys and optional relay.
- Import and export tools, and administration for profiles and music metadata.
- Themeable interface and support for existing Koito SQLite data.

## Roadmap

- [x] Simplify the architecture using Go, HTML, CSS and minimal JavaScript.
- [ ] Restructure administration, improve user separation and refine the UI.
- [ ] Make all configuration options available through `conf.toml` or the UI.
- [ ] Add multilingual support to the UI.
- [ ] Integrate listening history from Spotify, Deezer and Tidal, and review ListenBrainz for listens not submitted by TuneTrail.
- [ ] Generate images for social media posts, such as Instagram feeds.

## Installation

See [DEPLOYMENT.md](DEPLOYMENT.md) for setup and upgrades. The target image is `ghcr.io/ebrgr/tunetrail:0.7`, for Linux amd64 and arm64. Earlier SUM images retain their original tags. A matching image must be published before it can be pulled; use the local build instructions if it is unavailable.

```sh
cp deploy/.env.example deploy/.env
# Set the administrator password and data directory in deploy/.env.
docker compose --env-file deploy/.env -f deploy/compose.yaml pull
docker compose --env-file deploy/.env -f deploy/compose.yaml up -d
```

Back up your data before upgrading. Existing configuration names, the data directory, `koito.db`, API routes and Koito import format remain compatible. See the [profile and upgrade guide](docs/src/content/docs/guides/profiles.md) and [player configuration](docs/src/content/docs/guides/scrobbler.md).

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) and the [development roadmap](<backlog/docs/roadmaps/doc-001 - ROADMAP.md>). The current source version is **TuneTrail 0.7**; the 0.x version advances once per accepted task, when `make issue.publish` publishes it (for example, 0.9 to 0.10), and `make version.check` checks synchronized versions. Version 1.0 is reserved for a later decision.

## Credits

TuneTrail is based on Koito, created by Gabe Farrell, and builds on its listening history, API, catalog and import functionality. See [LICENSE](LICENSE) for license details.
