# Caching Tools

Self-hosted, local-first toolbox for geocaching, GPS and coordinate enthusiasts.

Caching Tools is aimed at people who use GPS as a hobby: geocachers, waypoint and track users, handheld-GPS users, and anyone who regularly works with coordinates in the field. It is not intended to depend on any single geocaching service or provider.

## Current status

M1 provides the first usable coordinate toolbox:

- DD / DMM / DMS parsing and conversion
- latitude/longitude and hemisphere validation
- great-circle distance calculation
- initial bearing calculation
- JSON endpoints for conversion and navigation
- interactive web UI
- Go unit and HTTP tests

Core tools work locally without a Geocaching.com account, API key or network provider.

## Run with Go

```sh
go run ./cmd/caching-tools
```

Open <http://localhost:8080>.

## Run with Docker/Podman Compose

```sh
docker compose up --build
```

Then open <http://localhost:8080>.

## Health check

```sh
curl http://localhost:8080/healthz
```

## Product direction

Caching Tools is intended to grow into a broad GPS and geocaching toolbox with:

- DD / DMM / DMS coordinate conversion
- distance, bearing, projection and intersection tools
- UTM/MGRS and other useful coordinate formats
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
- M1: coordinate conversion, distance and bearing
- M1.1: coordinate projection / destination point
- Next: UTM/MGRS, waypoint and GPX fundamentals

## License

MIT. See [LICENSE](LICENSE).
