# Caching Tools

Self-hosted, local-first toolbox for geocaching, GPS and coordinate enthusiasts.

Caching Tools is aimed at people who use GPS as a hobby: geocachers, waypoint and track users, handheld-GPS users, and anyone who regularly works with coordinates in the field. It is not intended to depend on any single geocaching service or provider.

## Current status

M1.38 provides a useful coordinate, grid, waypoint, GPX, map, field-navigation and geocaching puzzle toolbox:

- DD / DMM / DMS parsing and conversion
- great-circle distance, bearing and waypoint projection
- WGS84 ↔ UTM plus 1 m MGRS generation
- Norway and Svalbard UTM zone exceptions
- WGS84 (`EPSG:4326`) ↔ ETRS89 (`EPSG:4258`) geographic conversion with explicit approximation metadata
- ETRS89 / UTM zones 28N–38N (`EPSG:25828`–`EPSG:25838`) using GRS80
- local datum/CRS converter UI and `POST /api/coordinates/crs`
- field navigation to saved waypoints with distance and bearing
- nearest-point and cross-track distance to saved routes/tracks
- route/track along-distance, remaining distance and progress percentage
- next path point, distance to next point and forward bearing guidance
- configurable off-route threshold and arrival radius
- explicit `on-route`, `off-route`, `next-point-arrival`, `go-to` and `arrived` field statuses
- back-on-track guidance points to the nearest path position when outside the deviation threshold
- live browser navigation sessions using `watchPosition`
- continuous distance, bearing, progress and guidance refresh during live mode
- serialized live navigation requests so newer GPS positions replace stale pending updates
- in-memory session breadcrumbs with timestamp, coordinate and browser-reported accuracy
- configurable minimum breadcrumb distance, default 3 m
- configurable maximum accepted browser GPS accuracy, default 50 m
- accepted/rejected breadcrumb quality counters
- live session statistics for travelled distance, duration, average speed and maximum accepted segment speed
- defensive speed handling ignores invalid/non-positive time deltas and >360 km/h maximum-speed outliers
- pause/resume breadcrumb recording while live navigation continues
- resume starts a new recording segment so statistics and GPX do not bridge pause gaps
- manual in-memory field markers with `cache`, `trailhead`, `note` and custom marker types
- optional marker notes and current-position capture even while breadcrumb recording is paused
- explicit promotion of a selected session marker into the persistent local waypoint store
- promoted markers preserve coordinate, marker type, note and original marker timestamp provenance
- duplicate promotion of the same in-memory marker is prevented during the current page session
- promoted waypoints immediately refresh the waypoint list, navigation targets and local map
- persistent local field notes/logbook under `/data/field-notes.json`
- field-note title, body, status, type and occurrence timestamp
- optional soft links from field notes to saved waypoints and mystery workspaces
- field-note create/edit/delete UI and CRUD API
- browser-local field-note free-text search and status/type/waypoint/workspace filters
- explicit export of the currently filtered field-note set as versioned JSON or CSV
- field-note search/filter/export does not modify the persistent store or schema
- versioned field-note JSON backup import via `POST /api/field-notes/import`
- strict field-note bundle format/version validation
- imported field notes receive new local IDs and fresh create/update timestamps
- field-note restore appends atomically and never partially imports an invalid bundle
- existing notes remain unchanged during backup restore
- local logbook dashboard with total notes, last-7-day and last-30-day activity
- field-note status/type distributions and six-month UTC activity summary
- most-used linked waypoint and mystery-workspace summaries with local name resolution
- dashboard aggregation is browser-local and read-only with no external telemetry
- waypoint selection on the local map opens a waypoint-specific field-note view using stable waypoint IDs
- map waypoint logbook lists only notes whose `waypoint_id` matches the selected waypoint
- one-click creation of a new field note with the selected map waypoint preselected
- route/track map selections do not disturb the current waypoint logbook context
- map waypoint logbook refreshes after field-note CRUD/import changes
- local field-note attachments stored under `/data/field-note-attachments/<note-id>/`
- attachment upload/list/download/delete API for existing field notes
- 10 MiB per-file attachment limit with server-side byte sniffing and explicit content-type allowlist
- attachment filenames are sanitized and generated `att-*` IDs are path-traversal checked before filesystem access
- attachment downloads force `Content-Disposition: attachment` and `X-Content-Type-Options: nosniff`
- deleting a field note also removes its local attachment directory
- attachment UI supports field-note selection, upload, download, delete and refresh
- attachment-aware ZIP backup via `GET /api/field-notes/archive`
- ZIP archive keeps note/attachment metadata in `manifest.json` and binary attachments as separate archive entries
- attachment-aware restore via `POST /api/field-notes/archive` with 128 MiB expanded/archive guardrails
- archive restore rejects unsafe paths, duplicate/unreferenced entries and invalid attachment content before import
- restored notes and attachments receive new local IDs while note content, occurrence time and soft references are preserved
- restore appends without overwriting existing local objects and rolls back the newly created restore set if attachment creation fails
- SHA-256, preview kind, size and derived text/PDF metadata are exposed for local attachments
- existing older attachment sidecars remain compatible because derived metadata is recalculated from stored bytes
- safe local attachment previews via a separate `/preview` endpoint with `inline` disposition and `nosniff`
- image and PDF previews stay local; text-like previews are capped at 64 KiB
- attachment UI shows checksum, text line count/PDF version where applicable and an explicit Preview action
- explicit browser-side export of the current breadcrumb session as a GPX 1.1 track
- manual markers export as standard GPX 1.1 waypoints
- paused/resumed recording exports as separate GPX `<trkseg>` elements
- optional local GPX-export simplification with configurable tolerance, default 5 m
- simplification operates independently inside recording segments and leaves manual markers untouched
- timestamped `caching-tools-session-*.gpx` downloads
- breadcrumb export does not persist the session under `/data`
- breadcrumbs, markers and session statistics are cleared on new sessions or page reload unless explicitly exported or a marker is explicitly promoted
- track progress respects segment boundaries without bridging segment gaps
- optional one-shot browser geolocation as local field-navigation input
- map selection can become the field-navigation target
- bearing/bearing, bearing/distance and circle/circle intersections
- direct save of intersection solutions as local waypoints
- final-coordinate formula solver with variables A–Z
- persistent mystery workspaces under `/data/mystery-workspaces.json`
- workspace GC code/title, notes, A–Z variables and intermediate results
- workspace latitude/longitude final formulas and saved final-waypoint reference
- **Use in solver** loads workspace variables and formulas directly into the final-coordinate solver
- latest puzzle-helper result can be saved as a workspace intermediate result
- numeric puzzle-helper output can be assigned directly to an A–Z workspace variable
- saved final waypoints automatically link back to the active mystery workspace
- portable, versioned mystery-workspace JSON export/import
- optional final-waypoint inclusion in exported mystery bundles
- imported bundles create new local workspace/waypoint IDs without changing the persistent store format
- create/edit/delete mystery workspace UI and CRUD API
- A1Z26 / letter values and sums
- Caesar / ROT shifts and ROT47
- Morse encode/decode
- Bacon cipher encode/decode
- base conversion between bases 2 and 36
- telephone/keypad letter values
- digit sum and digital root/checksum
- configurable monoalphabetic substitution
- local waypoint create/list/edit/delete and persistence under `/data/waypoints.json`
- GPX 1.1 waypoint import/export
- GPX route/track inspection and persistent paths under `/data/paths.json`
- segment-aware distance, elevation, duration and speed statistics
- rename and GPX export for saved routes/tracks
- local route point deletion and reordering
- local track point deletion/reordering within segments
- split and merge track segments with persistent geometry updates
- visual route/track point selection directly on the local SVG map
- drag saved route/track points to new coordinates
- delete selected path points and split/merge track segments from the map editor
- existing elevation/timestamp point data preserved during geometry editing
- interactive local SVG map with pan, zoom, selection and list linkage
- no external map tiles, CDN, provider account, API key or network dependency for core tools
- interactive web UI and JSON/GPX APIs
- Go unit, HTTP and static-asset tests

See [docs/M1_38.md](docs/M1_38.md).

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
- M1.37: attachment-aware ZIP backup/restore with manifest validation, new local IDs and rollback on restore failure
- M1.38: attachment SHA-256 metadata and safe local image/PDF/text previews
- Next: richer map overlays, attachment integrity verification workflows, or additional well-defined CRS families

## License

MIT. See [LICENSE](LICENSE).
