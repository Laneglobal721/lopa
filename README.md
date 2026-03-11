# Lopa

Lopa is a **lightweight, centralized** network quality measurement and monitoring tool written in Go. It provides active probes (ICMP/TCP/UDP/TWAMP-light), passive interface statistics, and netlink-based monitoring (interface, IP, route changes), with a unified REST API and CLI.

## Architecture

- **lopad** — Daemon: runs the measurement engine, HTTP API, optional reflector (UDP echo + TWAMP-light), and netlink monitor. Start this first on each measurement node.
- **lopa** — CLI client: talks to lopad via HTTP to create tasks, list/show/stop/delete them, and manage monitors.

The CLI uses `http://127.0.0.1:8080` by default; set `LOPA_DAEMON_ADDR` or use `--daemon` to point to another lopad.

## Features

### Active measurement (ping, tcp, udp, twamp)

- **Ping** — ICMP latency, loss, jitter.
- **TCP** — TCP connect (TCPing) to a host:port.
- **UDP** — UDP probe to a reflector (echo); measures RTT/loss/jitter.
- **TWAMP** — TWAMP-light client to a standard Session-Reflector (e.g. port 862).

Modes: **count** (N packets), **duration** (run for T seconds), **continuous** (until stopped, with sliding-window stats).  
In continuous mode you can set loss/latency thresholds and an **alert webhook**; recovery notifications are also sent.

### Passive measurement

- Sample **interface counters** (bytes/packets in and out, errors, drops) at an interval.
- Modes: duration or continuous. No probe traffic.

### Monitor (netlink, Linux)

- **Interface** — Link up/down, name/MTU changes.
- **IP** — Address add/delete on interfaces.
- **Route** — Route add/delete (optional filter by table or destination CIDR).

Events can be stored in memory and sent to a **webhook URL** per task.

### Reflector (built into lopad)

- Generic UDP echo (default `:8081`) for UDP probe.
- TWAMP-light Session-Reflector (default `:862`) for TWAMP-light.  
Disable with `--no-reflector` or config.

## Build

```bash
# CLI client (talks to daemon)
go build -o lopa ./cmd/lopa

# Daemon (measurement engine + API + reflector + monitor)
go build -o lopad ./cmd/lopad
```

## Run

```bash
# 1. Start the daemon (e.g. on the measurement node)
./lopad
# Optional: --no-reflector, --no-monitor
# Config via env: LOPA_HTTP_ADDR, LOPA_LOG_LEVEL, etc.

# 2. Use the CLI (default daemon: http://127.0.0.1:8080)
./lopa --help
./lopa ping 192.168.1.1 --mode count --count 4
./lopa tcp example.com:443
./lopa udp reflector-host:8081 --mode duration --duration 30s
./lopa twamp reflector-host:862
./lopa passive eth0 --mode duration --duration 1m
./lopa task list
./lopa task show <task-id>
./lopa monitor add --type interface --interface eth0 --webhook-url http://localhost:9999/hook
./lopa monitor add --type route --route-dst 0.0.0.0/0
./lopa monitor list
./lopa monitor events <monitor-id>
```

## API (HTTP)

Base path: `/api/v1`.

| Method | Path | Description |
|--------|------|-------------|
| POST | /tasks/ping, /tasks/tcp, /tasks/udp, /tasks/twamp, /tasks/passive | Create measurement task |
| GET | /tasks | List all tasks |
| GET | /tasks/:id | Get task result |
| POST | /tasks/:id/stop | Stop task |
| DELETE | /tasks/:id | Delete task |
| POST | /monitors | Create monitor task |
| GET | /monitors | List monitors |
| GET | /monitors/:id | Get monitor |
| PATCH | /monitors/:id | Update monitor |
| DELETE | /monitors/:id | Delete monitor |
| GET | /monitors/:id/events?last=N | Get recent events |

## Configuration

lopad uses [Viper](https://github.com/spf13/viper): config file, environment variables (`LOPA_*`), or defaults.

| Key | Default | Description |
|-----|---------|-------------|
| http.addr | :8080 | HTTP API listen address |
| log.level | info | Log level |
| reflector.enabled | true | Enable reflector |
| reflector.addr | :8081 | UDP echo listen address |
| reflector.twamp_addr | :862 | TWAMP-light reflector; empty to disable |
| monitor.enabled | true | Enable netlink monitor |
| monitor.event_buffer_size | 100 | Max events per monitor task |

## License

See repository. This project is under active development.
