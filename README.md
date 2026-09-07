# Caching Tools

Self-hosted, local-first toolbox for geocaching, GPS and coordinate enthusiasts.

Caching Tools is aimed at people who use GPS as a hobby: geocachers, waypoint and track users, handheld-GPS users, and anyone who regularly works with coordinates in the field. It is not intended to depend on any single geocaching service or provider.

## Current status

M1.18 provides a useful coordinate, grid, waypoint, GPX, map and geocaching puzzle toolbox:

- DD / DMM / DMS parsing and conversion
- great-circle distance, bearing and waypoint projection
- WGS84 ↔ UTM plus 1 m MGRS generation
- Norway and Svalbard UTM zone exceptions
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
- interactive local SVG map with pan, zoom, selection and list linkage
- no external map tiles, CDN, provider account, API key or network dependency for core tools
- interactive web UI and JSON/GPX APIs
- Go unit, HTTP and static-asset tests

See [docs/M1_18.md](docs/M1_18.md).

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

The compose volume mounted at `/data` keeps saved waypoints, routes, tracks and mystery workspaces across container recreation.

## Product direction

Caching Tools is intended to grow into a broad GPS and geocaching toolbox with coordinate tools, waypoint management, GPX workflows, track/route analysis, offline/local visualization, mystery-cache helpers, final-coordinate solving, local notes/logbook data, datum conversion and optional provider integrations.

Core functionality should remain useful offline and without API keys. Private cache, waypoint, track and field-note data should remain local by default.

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
- Next: richer route/track editing or workspace bundle evolution

## License

MIT. See [LICENSE](LICENSE).
