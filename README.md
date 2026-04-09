# About



# Quick start (`docker` and `docker compose` required)

- `docker compose up -d`
- Go to `localhost:8080`
- Test yourself. Also open in several tabs and try to modify packs / calculate concurrently

# Test locally (`go` and `make` installed nativelly required)

- `make test` - runs unit tests
- `make bench` - runs main algo benchmark

## Curl examples:

- Get current packs - `curl -s localhost:8080/api/v1/packs | jq`
- Set custom packs  - `curl -s -X POST localhost:8080/api/v1/packs -d '{"packs": [23, 31, 53]}' | jq`
- Calculate         - `curl -s -X POST localhost:8080/api/v1/calculate -d '{"items": 1}' | jq`
- Reset packs       - `curl -s -X POST localhost:8080/api/v1/packs -d '{"packs": [250, 500, 1000, 2000, 5000]}' | jq`