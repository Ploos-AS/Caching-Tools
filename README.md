# Caching Tools

Self-hosted, local-first toolbox for geocaching, GPS and coordinate enthusiasts.

Caching Tools is aimed at people who use GPS as a hobby: geocachers, waypoint and track users, handheld-GPS users, and anyone who regularly works with coordinates in the field. It is not intended to depend on any single geocaching service or provider.

## Current status

M1.45 provides a useful coordinate, grid, waypoint, GPX, map, field-navigation and geocaching puzzle toolbox. Highlights include:

- DD / DMM / DMS, WGS84/UTM/MGRS and WGS84/ETRS89 coordinate tools
- distance, bearing, projection and coordinate-intersection tools
- persistent local waypoints plus GPX 1.1 import/export
- persistent routes/tracks with statistics and geometry editing
- interactive offline SVG map with pan, zoom, selection and visual route/track editing
- optional local map overlays for waypoint labels and configurable waypoint-radius rings
- current field-navigation position marker and selected-waypoint guidance line on the SVG map
- selected waypoint arrival-radius overlay tied to the field-navigation arrival-radius setting
- route/track navigation overlays for current-to-nearest-path, nearest point, next point and forward segment
- route/track overlay status for `on-route`, `off-route`, `next-point-arrival` and `arrived`
- map overlay readout for cross-track distance, remaining distance and forward bearing
- stable-ID map/list linkage: map selections resolve list rows by object kind + local ID, not display names or rendered text
- waypoint/path list rows are automatically re-annotated after CRUD rerenders, so duplicate names cannot select the wrong row
- hardened dynamic browser asset chain with readiness sentinels, idempotent continuation and failed-load retry
- executable Node runtime tests for loader state transitions, retry, duplicate-safe map/list selection and window-load orchestration
- executable runtime tests for waypoint guidance overlays, route/track cross-track/next-point overlays and stale-overlay clearing
- browser runtime qualification uses Node's built-in `node:test` and small local fake-DOM harnesses; no npm dependencies are required
- field navigation to waypoints and nearest/cross-track/progress guidance for saved paths
- live browser geolocation navigation with quality-filtered, segment-aware breadcrumbs
- explicit GPX export of live sessions and promotion of manual field markers to waypoints
- persistent mystery workspaces, final-coordinate formulas and provider-neutral puzzle helpers
- persistent local field notes/logbook with search/filter/export and map waypoint linkage
- local field-note attachments with 10 MiB limit, type validation, safe download/preview and cascade cleanup
- attachment-aware ZIP backup/restore with validation, new IDs and rollback semantics
- SHA-256 attachment metadata plus explicit read-only integrity verification
- local attachment integrity UI/report with `ok`, `mismatch`, `missing` and `unrecorded` states
- no external map tiles, CDN, provider account, API key or network dependency for core tools
- one OCI-oriented Go application with local persistence under `/data`
- Go unit, HTTP/static-asset tests and browser-side runtime tests

See [docs/M1_45.md](docs/M1_45.md). Loader/list runtime qualification details are in [docs/M1_44.md](docs/M1_44.md).

## Run with Go

```sh
go run ./cmd/caching-tools
```

Open <http://localhost:8080>.

For local development, set `CACHING_TOOLS_DATA_DIR` if you do not want persistent data under `/data`.

## Run with Docker/Podman Compose

```sh
docker compose up --build
```

Then open <http://localhost:8080>.

The compose volume mounted at `/data` keeps saved waypoints, routes, tracks, field notes, field-note attachments and mystery workspaces across container recreation.

## Product direction

Caching Tools is intended to grow into a broad GPS and geocaching toolbox with coordinate tools, waypoint management, GPX workflows, track/route analysis, offline/local visualization, mystery-cache helpers, final-coordinate solving, local notes/logbook data, datum conversion and optional provider integrations.

Core functionality should remain useful offline and without API keys. Private cache, waypoint, track, field-note and attachment data should remain local by default.

## Milestones

- M0: runnable Go/OCI foundation
- M1: DD/DMM/DMS conversion, distance and bearing
- M1.1: waypoint projection / destination point
- M1.2: WGS84/UTM conversion and MGRS output
- M1.3: local persistent waypoint CRUD
- M1.4: GPX 1.1 waypoint import/export
- M1.5: GPX track and route inspection
- M1.6: elevation/time-aware track statistics
- M1.7: persistent route and track objects
- M1.8: rename and GPX export for saved routes/tracks
- M1.9: local SVG map for saved GPS data
- M1.10: interactive map navigation, selection and list linkage
- M1.11: bearing/bearing, bearing/distance and circle/circle intersections
- M1.12: save intersection solutions as persistent waypoints
- M1.13: geocaching final-coordinate formula solver
- M1.14: A1Z26, Caesar/ROT, digit checksum and substitution puzzle helpers
- M1.15: Morse, Bacon, ROT47, base conversion and phone/keypad puzzle helpers
- M1.16: persistent mystery workspaces with notes, variables, formulas and final-waypoint linkage
- M1.17: active-workspace integration with solver, puzzle results and automatic final-waypoint linking
- M1.18: portable mystery workspace JSON bundles with optional final-waypoint roundtrip
- M1.19: persistent local route/track geometry editing
- M1.20: visual route/track editing directly on the local SVG map
- M1.21: WGS84/ETRS89 datum handling and explicit ETRS89 / UTM CRS conversion
- M1.22: local field navigation to waypoints and nearest/cross-track navigation for saved paths
- M1.23: along-path progress, remaining distance, next-point distance and forward-bearing guidance
- M1.24: configurable route-deviation alerts, back-on-track status and arrival guidance
- M1.25: live geolocation navigation sessions with continuous guidance and ephemeral breadcrumbs
- M1.26: explicit GPX 1.1 export of the current in-memory live-navigation breadcrumb session
- M1.27: live-session distance, duration, average-speed and maximum-speed statistics
- M1.28: breadcrumb minimum-distance and accuracy filters plus optional GPX-export simplification
- M1.29: pause/resume breadcrumb recording, segment-aware session statistics/GPX and manual field markers
- M1.30: explicit promotion of manual field-session markers into persistent local waypoints
- M1.31: persistent local field notes/logbook with optional waypoint and mystery-workspace links
- M1.32: browser-local field-note search/filter plus explicit filtered JSON/CSV export
- M1.33: versioned field-note JSON backup import with new local IDs and atomic append restore
- M1.34: local read-only logbook dashboard with activity, status/type and linked-target summaries
- M1.35: map waypoint logbook integration and prefilled field-note creation from map selection
- M1.36: local field-note attachments with size/type validation, download hardening and cascade cleanup
- M1.37: attachment-aware ZIP backup/restore with manifest validation, new IDs and rollback on restore failure
- M1.38: attachment SHA-256 metadata and safe local image/PDF/text previews
- M1.39: explicit read-only attachment integrity verification and downloadable local report
- M1.40: richer local GPS map overlays with labels, radius rings and waypoint guidance visualization
- M1.41: route/track navigation overlays for nearest path, next point, forward guidance and status
- M1.42: stable-ID map/list linkage hardening for duplicate-safe waypoint and path selection
- M1.43: idempotent dynamic asset loading with readiness detection and failed-load retry
- M1.44: executable browser-side runtime qualification for dynamic UI orchestration and stable-ID selection
- M1.45: executable browser runtime qualification for waypoint/path navigation map overlays and rerender behavior
- Next: additional well-defined CRS families, field-navigation map polish, or broader runtime coverage for live session controls

## License

MIT. See [LICENSE](LICENSE).
