# Caching Tools

Self-hosted, local-first tools for geocaching, coordinates and puzzle solving.

## M0 foundation

M0 establishes the runnable project baseline:

- Go backend with embedded web UI
- `GET /healthz` health endpoint
- minimal multi-stage OCI image
- non-root runtime user
- persistent `/data` volume reserved for future SQLite/cache data
- Docker Compose example
- amd64/arm64-friendly source layout
- MIT licensed

The application does not require a Geocaching.com account, API key or network
provider at M0.

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

## Direction

Caching Tools is intended to grow into a toolbox for:

- DD / DMM / DMS coordinate conversion
- distance, bearing, projection and intersection tools
- UTM/MGRS and other useful coordinate formats
- mystery-cache cipher and number tools
- formula-based final coordinate solving
- GPX import/export
- local cache and waypoint database
- private field notes and logbook
- optional provider integrations

Core functionality should remain useful without API keys, and private cache,
waypoint and field-note data should remain local by default.

## License

MIT. See [LICENSE](LICENSE).
