# Pokemon Showdown LAN Setup

This workspace contains:

- Server: `/home/ysu/projects/showdown-suite/pokemon-showdown-local`
- Client: `/home/ysu/projects/showdown-suite/pokemon-showdown-client-local`

## What is configured

- The server listens on `0.0.0.0:8000` by default.
- The local client is served on `0.0.0.0:8080` by default.
- `config/config.js` enables `noguestsecurity`, so LAN users can pick names without the public login server.
- SQLite-backed local features are enabled for PMs, friends, and modlog.
- The server root page links directly to the local client URL.
- `[Gen 9] Showdown Suite Studio` is loaded from `config/showdown-suite-local-format.json`.

## Start

```bash
cd /home/ysu/projects/showdown-suite/pokemon-showdown-local
./scripts/start-lan-stack.sh
```

## Stop

```bash
cd /home/ysu/projects/showdown-suite/pokemon-showdown-local
./scripts/stop-lan-stack.sh
```

To restart only the server after changing the local studio format:

```bash
cd /home/ysu/projects/showdown-suite/pokemon-showdown-local
./scripts/restart-lan-server.sh
```

## Status

```bash
cd /home/ysu/projects/showdown-suite/pokemon-showdown-local
./scripts/status-lan-stack.sh
```

## Admin account

To grant admin to a local username:

```bash
cd /home/ysu/projects/showdown-suite/pokemon-showdown-local
./scripts/create-admin.sh yourname
```

Then reconnect with that username.

## LAN client URL

Replace the IP if your machine has a different LAN address:

```text
http://YOUR_LAN_IP:8080/play.pokemonshowdown.com/testclient.html?~~YOUR_LAN_IP:8000
```

The server landing page at `http://YOUR_LAN_IP:8000` also links to the correct client URL.

## Notes for AI work

- For battle automation and co-op agents, connect to the running server over the Showdown websocket protocol.
- Keep the LAN server local-only while you iterate on AI behavior.
- If you later want browser login, registrations, or internet exposure, that is a separate login-server and TLS job.
- Use `config/showdown-suite-local-format.json` or the Go GUI to adjust the local studio format, extra Showdown rules, target Pokémon pool, and extra banned Pokémon.
