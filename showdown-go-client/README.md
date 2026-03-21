# Showdown Go Client

`showdown-go-client` is a dedicated Go project for:

- embedding Pokémon Showdown connectivity into other Go programs
- calling features from the CLI
- using a local GUI control panel for connection checks, room status, and simple mock battles

## Project layout

- `pkg/showdown`: embeddable Go client and higher-level helpers
- `cmd/showcli`: CLI entrypoint
- `internal/httpapi`: local JSON API for other apps and the GUI
- `internal/gui`: browser GUI served by the same Go application
- `../pokemon-showdown-local/config/showdown-suite-local-format.json`: editable source for the local studio format

## Quick start

Start your local Showdown server first:

```bash
cd /home/ysu/projects/showdown-suite/pokemon-showdown-local
./scripts/start-lan-stack.sh
```

Then use this client:

```bash
cd /home/ysu/projects/showdown-suite/showdown-go-client
go run ./cmd/showcli ping
go run ./cmd/showcli status
go run ./cmd/showcli mockbattle --timeout 90s
go run ./cmd/showcli gui
```

## CLI examples

```bash
go run ./cmd/showcli ping --server http://127.0.0.1:8000 --username koharu
go run ./cmd/showcli status --server http://127.0.0.1:8000
go run ./cmd/showcli mockbattle --server http://127.0.0.1:8000 --format gen9randombattle --timeout 90s
go run ./cmd/showcli gui --addr 127.0.0.1:8099
```

## AI training client

This repository also includes `showtrain`, a separate CLI for self-play reinforcement learning from another PC.

Examples:

```bash
go run ./cmd/showtrain probe --server http://127.0.0.1:8000
go run ./cmd/showtrain train --server http://127.0.0.1:8000 --format gen9randombattle --battles 50
go run ./cmd/showtrain evaluate --server http://127.0.0.1:8000 --model models/selfplay-latest.json --battles 20
```

For custom teams on `[Gen 9] Showdown Suite Studio`, expose the Go API and pass `--api-base`. The Japanese guide is in `AI_TRAINING.ja.md`.

## HTTP API

When running `showcli serve` or `showcli gui`, the following local endpoints are available:

- `GET /api/local-format`
- `POST /api/local-format`
- `POST /api/ping`
- `POST /api/status`
- `POST /api/validate-team`
- `POST /api/mock-battle`

Example:

```bash
curl -s http://127.0.0.1:8099/api/status \
  -H 'Content-Type: application/json' \
  -d '{"server_url":"http://127.0.0.1:8000","username":"api"}'
```

The GUI can also edit the local studio format, including target Pokémon, extra banned species, and arbitrary additional Showdown rules, restart the LAN server, validate exported teams against `[Gen 9] Showdown Suite Studio`, and run mock battles with those validated teams.

## Notes

- This project intentionally uses Showdown's default websocket and challenge flow.
- The mock battle feature uses two local clients that challenge and accept on `randombattle` style formats.
- If the timeout is reached after the battle has started, the API returns a partial battle result instead of failing hard.
- No custom Showdown server modifications are required for the current feature set.
