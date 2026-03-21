# Showdown Suite

Japanese version: [README.ja.md](./README.ja.md)

`showdown-suite` is a workspace that combines:

- a local Pokémon Showdown server for LAN play
- a local copy of the Showdown web client
- a Go library, CLI, HTTP API, and browser GUI for automation and operations

This repository root is only a workspace wrapper. Each subdirectory keeps its own build tooling and dependencies.

## Workspace layout

- `pokemon-showdown-local`
  Local Showdown server plus LAN helper scripts and the local studio format definition.
- `pokemon-showdown-client-local`
  Local copy of the Showdown browser client used by the LAN stack.
- `showdown-go-client`
  Go package, CLI, HTTP API, and browser GUI for connecting to the local server.

## Requirements

- Go `1.25+`
- Node.js `16+`
- `python3` for serving the local browser client

Install dependencies inside the subprojects you plan to use:

```bash
cd pokemon-showdown-local && npm install
cd ../pokemon-showdown-client-local && npm install
cd ../showdown-go-client && go mod download
```

## Quick start

### 1. Start the LAN stack

```bash
cd pokemon-showdown-local
./scripts/start-lan-stack.sh
```

Useful companion commands:

```bash
./scripts/status-lan-stack.sh
./scripts/stop-lan-stack.sh
./scripts/create-admin.sh yourname
```

Default ports:

- Showdown server: `8000`
- Static web client: `8080`

`start-lan-server.sh` builds the server bundle automatically if `dist/` is missing.  
`start-lan-client.sh` serves the browser client from `pokemon-showdown-client-local` with `python3 -m http.server`.

### 2. Open the game client

Use either:

- the server landing page: `http://HOST:8000`
- the direct test client: `http://HOST:8080/play.pokemonshowdown.com/testclient.html?~~HOST:8000`

`./scripts/start-lan-stack.sh` prints a LAN-friendly URL based on the current machine IP when possible.

### 3. Run the Go tooling

```bash
cd showdown-go-client
go run ./cmd/showcli ping --server http://127.0.0.1:8000
go run ./cmd/showcli status --server http://127.0.0.1:8000
go run ./cmd/showcli gui --addr 127.0.0.1:8099
```

The GUI and API are served from `http://127.0.0.1:8099/assets/` by default.

## Go client overview

Main directories in `showdown-go-client`:

- `pkg/showdown`
  Embeddable Go client and higher-level helpers.
- `cmd/showcli`
  CLI entrypoint.
- `internal/httpapi`
  Local JSON API used by other apps and the GUI.
- `internal/gui`
  Browser GUI assets and HTTP handler wiring.
- `../pokemon-showdown-local/config/showdown-suite-local-format.json`
  Source-of-truth file for the local studio format configuration.

### CLI commands

`showcli` currently supports:

- `ping`
  Checks websocket connectivity and rename flow.
- `status`
  Fetches server connection info, room data, formats, and the local studio format definition.
- `mockbattle`
  Runs a simple automated battle between two local clients.
- `serve`
  Starts the local HTTP API and GUI without opening a browser.
- `gui`
  Starts the same local HTTP API and GUI and opens the browser automatically.

Examples:

```bash
go run ./cmd/showcli ping --server http://127.0.0.1:8000 --username koharu
go run ./cmd/showcli status --server http://127.0.0.1:8000 --username koharu
go run ./cmd/showcli mockbattle --server http://127.0.0.1:8000 --format gen9randombattle --timeout 90s
go run ./cmd/showcli serve --addr 127.0.0.1:8099
go run ./cmd/showcli gui --addr 127.0.0.1:8099
```

Note: custom teams for studio-format validation and mock battles are currently exposed through the HTTP API and GUI, not through dedicated `showcli mockbattle` flags.

## HTTP API

When `showcli serve` or `showcli gui` is running, these endpoints are available:

- `GET /api/healthz`
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

`POST /api/local-format` updates the studio format configuration. Species names in `targetPokemon` and `bannedPokemon` are canonicalized against Showdown data, and unknown names are rejected with an explicit error.

## Local studio format

The local studio format is the format named `[Gen 9] Showdown Suite Studio`.

Its editable configuration lives in:

- `pokemon-showdown-local/config/showdown-suite-local-format.json`

You can edit it in two ways:

- from the Go GUI
- by calling `POST /api/local-format`

Supported settings include:

- preset selection for singles or doubles
- level, team size, and picked team size
- open team sheets
- terastallization toggle
- `targetPokemon`
- `bannedPokemon`
- additional Showdown custom rules

The server distributes the resulting rule definition through the local-format query/API path, and team validation rejects illegal teams with explicit messages.

## Additional docs

- LAN-specific notes: `pokemon-showdown-local/LOCAL_LAN_SETUP.md`
- Japanese handoff guide: `HANDOFF.ja.md`
- Go client details: `showdown-go-client/README.md`
- Upstream Showdown server documentation: `pokemon-showdown-local/README.md`
- Upstream Showdown client documentation: `pokemon-showdown-client-local/README.md`
