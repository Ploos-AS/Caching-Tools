# Caching Tools

Self-hosted, local-first toolbox for geocaching, GPS and coordinate enthusiasts.

Caching Tools is aimed at people who use GPS as a hobby: geocachers, waypoint and track users, handheld-GPS users, and anyone who regularly works with coordinates in the field. It is not intended to depend on any single geocaching service or provider.

## Current status

M1.5 provides a useful coordinate, grid, waypoint and GPX toolbox:

- DD / DMM / DMS parsing and conversion
- latitude/longitude and hemisphere validation
- great-circle distance and initial bearing
- waypoint projection from bearing and distance
- WGS84 → UTM conversion
- UTM → WGS84 conversion
- 1 m precision MGRS generation
- Norway and Svalbard UTM zone exceptions
- local waypoint create/list/edit/delete
- waypoint persistence under `/data`
- GPX 1.1 waypoint import/export
- GPX 1.1 route inspection with point count and distance
- GPX 1.1 track inspection with segment count, point count and distance
- segment-aware track distance calculation
- interactive web UI and JSON/GPX APIs
- Go unit and HTTP tests

Core tools work locally without a Geocaching.com account, API key or network provider. See [docs/M1_5.md](docs/M1_5.md).

## Run with Go

```sh
go run ./cmd/caching-tools
```

Open <http://localhost:8080>.

For local development, set `CACHING_TOOLS_DATA_DIR` if you do not want waypoint data under `/data`.

## Run with Docker/Podman Compose

```sh
docker compose up --build
```

Then open <http://localhost:8080>.

The compose volume mounted at `/data` keeps saved and imported waypoints across container recreation.

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
- Next: elevation/time-aware track statistics and persistent track/route objects

## License

MIT. See [LICENSE](LICENSE).
