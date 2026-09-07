# Caching Tools

Self-hosted, local-first toolbox for geocaching, GPS and coordinate enthusiasts.

Caching Tools is aimed at people who use GPS as a hobby: geocachers, waypoint and track users, handheld-GPS users, and anyone who regularly works with coordinates in the field. It is not intended to depend on any single geocaching service or provider.

## Current status

M1.11 provides a useful coordinate, grid, waypoint, GPX and interactive local map toolbox:

- DD / DMM / DMS parsing and conversion
- latitude/longitude and hemisphere validation
- great-circle distance and initial bearing
- waypoint projection from bearing and distance
- bearing/bearing coordinate intersections
- bearing/distance ray-circle intersections
- circle/circle coordinate intersections
- WGS84 → UTM conversion
- UTM → WGS84 conversion
- 1 m precision MGRS generation
- Norway and Svalbard UTM zone exceptions
- local waypoint create/list/edit/delete
- waypoint persistence under `/data/waypoints.json`
- GPX 1.1 waypoint import/export
- GPX 1.1 route and track inspection
- segment-aware track distance calculation
- elevation gain/loss, duration and speed statistics
- persistent routes and tracks under `/data/paths.json`
- full route point and track segment preservation
- rename and GPX export for saved routes/tracks
- local SVG map for saved waypoints, routes and tracks
- drag-to-pan and wheel/button zoom
- fit-all and fit-to-selection map controls
- clickable and keyboard-selectable map objects
- map selection linked to the corresponding saved-data list row
- automatic map bounds and preserved track segment gaps
- no external map tiles, CDN, map API key or provider dependency
- interactive web UI and JSON/GPX APIs
- Go unit and HTTP/static-asset tests

Core tools work locally without a Geocaching.com account, API key or network provider. See [docs/M1_11.md](docs/M1_11.md).

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

The compose volume mounted at `/data` keeps saved waypoints, routes and tracks across container recreation.

## Health check

```sh
curl http://localhost:8080/healthz
```

## Product direction

Caching Tools is intended to grow into a broad GPS and geocaching toolbox with:

- coordinate conversion across common GPS/map formats
- distance, bearing, projection and intersection tools
- waypoint creation and management
- GPX import, inspection, editing and export
- track and route analysis
- elevation, speed and distance statistics
- offline/local map visualization with optional configured basemaps
- mystery-cache cipher and number tools
- formula-based final coordinate solving
- local cache and waypoint database
- private field notes and logbook
- datum and coordinate-system conversion
- optional provider integrations

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
- Next: save intersection results as waypoints and richer route/track editing

## License

MIT. See [LICENSE](LICENSE).
