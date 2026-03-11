# Implementation Plan: Monitor Tasks & Passive Measurement

## Overview

This document evaluates two proposed features and provides a phased implementation plan:

1. **Monitor tasks** – Netlink-based monitoring (interface, IP, route changes) with delivery via API and `lopa monitor` CLI.
2. **Passive measurement** – Statistics and counting without sending probe traffic (e.g. interface counters).

---

## Feature 1: Monitor Tasks (Netlink-Based)

### 1.1 Description

- Users create **monitor tasks** via API (and optionally CLI `lopa monitor`).
- **lopad** subscribes to netlink and detects:
  - **Interface**: link up/down, name/MTU change, interface add/delete.
  - **IP**: address add/delete on interfaces (IPv4/IPv6).
  - **Route**: routing table changes (route add/delete/change).
- When a change occurs that matches a task, lopad **reports** to the user (webhook and/or stored events, queryable via API).

### 1.2 Evaluation

| Aspect | Assessment |
|--------|------------|
| **Feasibility** | High. Go has solid netlink support (e.g. `vishvananda/netlink`). Events are well-defined. |
| **Platform** | Linux-only. Netlink is Linux-specific; other OS would need a different backend or stub. |
| **Privileges** | May require root or `CAP_NET_ADMIN` for full route/link subscription; interface list and stats often possible without. |
| **Complexity** | Medium. Need a single netlink listener goroutine, task filter matching, and event delivery. |
| **Fit with Lopa** | Fits well: same daemon (lopad), new “monitor” domain next to “measurement” tasks; REST API and CLI stay consistent. |

### 1.3 Scope Recommendation

- **Phase 1 (MVP)**  
  - Modes: **interface** (link state, optional filter by name/index), **ip** (address add/delete, optional filter by interface or prefix).  
  - One netlink listener in lopad, in-memory list of monitor tasks, event payload (type, interface, summary).  
  - Delivery: **webhook URL** per task (POST JSON on match). Optional: store last N events per task in memory and expose via API.

- **Phase 2**  
  - **Route** changes (with clear filtering to avoid flood).  
  - Optional: event history API (paginated), retention limit.

- **Out of scope for now**  
  - ARP/NDP table monitoring, conntrack, traffic sampling.

### 1.4 Data Model (Draft)

```text
MonitorTask
  - id: string (same pattern as measurement task id)
  - type: "interface" | "ip" | "route"
  - filter: (e.g. interface_name, interface_index, prefix, route_dst)
  - webhook_url: string (optional)
  - enabled: bool
  - created_at, updated_at

MonitorEvent (when a change is detected)
  - id: uuid
  - task_id: string
  - type: "interface" | "ip" | "route"
  - change: "add" | "delete" | "update"
  - detail: JSON (interface name, index, flags; address; route dst/gw; etc.)
  - at: timestamp
```

### 1.5 API Design (REST)

- `POST   /api/v1/monitors`          – Create monitor task (body: type, filter, webhook_url).
- `GET    /api/v1/monitors`          – List monitor tasks.
- `GET    /api/v1/monitors/:id`      – Get one task.
- `PATCH  /api/v1/monitors/:id`     – Update (e.g. enable/disable, webhook).
- `DELETE /api/v1/monitors/:id`     – Delete task.
- `GET    /api/v1/monitors/:id/events` – (Optional) Last N events for this task (if stored).

### 1.6 CLI Design (`lopa monitor`)

- `lopa monitor list`                    – List monitor tasks (call API).
- `lopa monitor add --type interface [--interface eth0] [--webhook-url URL]`
- `lopa monitor add --type ip [--interface eth0] [--webhook-url URL]`
- `lopa monitor add --type route [--webhook-url URL]`   (Phase 2)
- `lopa monitor delete <id>`
- `lopa monitor events <id> [--last N]`  – (Optional) Show recent events.

### 1.7 Implementation Plan (Feature 1)

1. **Internal packages**
   - `internal/monitor`: task definition, in-memory store, filter matching.
   - `internal/monitor/netlink`: netlink listener (link + addr in Phase 1; route in Phase 2), emit generic “change” structs.

2. **lopad**
   - On startup (if config permits): start netlink listener goroutine; no dependency on measurement engine.
   - Listener receives netlink messages → normalizes to “MonitorEvent” → for each registered MonitorTask, if filter matches → POST to task’s webhook and optionally append to in-memory event ring.

3. **HTTP API**
   - Register `/api/v1/monitors` routes; handlers CRUD on monitor store and optionally event buffer.

4. **CLI**
   - New `internal/cli/monitor.go`: `monitor` subcommand; all operations via existing daemon base URL (same as tasks).

5. **Config**
   - `monitor.enabled` (default true); `monitor.event_buffer_size` (e.g. 100 per task) if we store events.

6. **Dependencies**
   - Add e.g. `github.com/vishvananda/netlink` (and possibly `netlink/nl` for route groups if needed); document Linux-only.

---

## Feature 2: Passive Measurement

### 2.1 Description

- **Passive measurement** = observe existing traffic or system counters **without sending probe packets**.
- Users create **passive measurement tasks** that define:
  - **What** to observe (e.g. interface counters: bytes in/out, packets in/out, errors/drops).
  - **How often** to sample (interval).
  - **How long** or **how many** samples (mode: count vs duration vs continuous, similar to active tasks).
- lopad **polls** sources (e.g. netlink stats or `/proc/net/dev`), aggregates, and exposes results via API and optional webhook.

### 2.2 Evaluation

| Aspect | Assessment |
|--------|------------|
| **Feasibility** | High for interface counters (netlink or /proc); no extra packet injection. |
| **Platform** | Interface stats: Linux (netlink or /proc); other OS could use different backends later. |
| **Privileges** | Usually no special privileges for interface stats. |
| **Complexity** | Medium. Polling loop, counter deltas, optional time-series storage and retention. |
| **Fit with Lopa** | Fits: “passive” is another task type alongside ping/tcp/udp/twamp; same result/stat concepts (counts, rates). |

### 2.3 Scope Recommendation

- **Phase 1 (MVP)**  
  - **Source**: interface(s) only (by name).  
  - **Metrics**: bytes in/out, packets in/out (and optionally errors/drops if easily available).  
  - **Modes**: **duration** (sample every N sec for T seconds) and **continuous** (ongoing until stop).  
  - **Output**: same unified result style – e.g. `Total` with `Sent`/`Received` repurposed as bytes_in/bytes_out and packet counts; optional `Rounds` as fixed windows.  
  - **Storage**: in-memory result per task; no long-term time-series DB in MVP (optional: keep last K samples in memory for “recent history” API).

- **Phase 2**  
  - **Rates** (bytes/sec, packets/sec) in addition to counters.  
  - Optional **threshold + webhook** (e.g. when rate &gt; X or errors &gt; Y).  
  - Optional **lightweight retention** (e.g. last 1h of points in memory with downsampling).

- **Out of scope for now**  
  - Flow-level passive (sFlow/NetFlow), deep packet inspection, BPF-based custom metrics.

### 2.4 Data Model (Draft)

```text
PassiveTaskParams (creation)
  - type: "passive"
  - source: "interface"
  - target: string (e.g. "eth0" or "eth0,eth1" for multi)
  - interval: duration (e.g. 10s)
  - mode: "duration" | "continuous"
  - duration: duration (for duration mode)
  - metrics: ["bytes_in","bytes_out","packets_in","packets_out"] (optional; default all)

PassiveResult (aligned with existing Result where possible)
  - task_id, node_id, started, finished, status
  - total: { bytes_in, bytes_out, packets_in, packets_out, errors_in, errors_out, ... }
  - rounds: [ { from, to, stats } ]  (each round = one aggregation window)
  - window: (for continuous) sliding window stats
```

### 2.5 API Design (REST)

- Reuse existing task API where possible:
  - `POST   /api/v1/tasks/passive` – Create passive task (body: PassiveTaskParams).
  - `GET    /api/v1/tasks` – Include passive tasks in list.
  - `GET    /api/v1/tasks/:id` – Return passive result (same endpoint as active tasks).
  - `POST   /api/v1/tasks/:id/stop` – Stop passive task.
  - `DELETE /api/v1/tasks/:id` – Delete task and result.

- Optional later:
  - `GET    /api/v1/tasks/:id/history` – Last N samples (if we keep in-memory history).

### 2.6 CLI Design

- `lopa passive eth0 [--interval 10s] [--mode duration|continuous] [--duration 5m]`  
  - Creates passive task via API and optionally streams or waits for result (same pattern as `lopa ping`/`lopa tcp`).
- `lopa task list` / `lopa task show <id>` – Already show all tasks; passive tasks appear with type "passive".

### 2.7 Implementation Plan (Feature 2)

1. **Internal packages**
   - `internal/passive`: task params, result struct, aggregator (compute deltas and window stats).
   - `internal/passive/source`: interface to “fetch current counters” (e.g. `InterfaceStats(iface string) (bytesIn, bytesOut, packetsIn, packetsOut, ...)`).
   - Implementation of source: use `vishvananda/netlink` (link list + stats) or parse `/proc/net/dev` (no extra dep).

2. **Engine**
   - Add `CreatePassiveTask(params)` in measurement engine (or a dedicated passive engine that reuses same store pattern).
   - Run loop: tick every `interval`, read interface stats, compute deltas from previous sample, update `Result.Total` and append to `Rounds` or update `Window` for continuous; respect `duration` and `stop`.

3. **Unified result**
   - Extend `Result` (or add a `PassiveResult` embed) with optional fields for bytes_in/out, packets_in/out so that `GET /api/v1/tasks/:id` returns one schema; CLI can display “bytes_in, bytes_out, packets_in, packets_out” for type passive.

4. **HTTP API**
   - `POST /api/v1/tasks/passive` handler; reuse list/get/stop/delete for tasks.

5. **CLI**
   - New `internal/cli/passive.go`: `passive [interface]` with flags; create task via API and poll/stream result like other task types.

6. **Config**
   - No mandatory config for MVP; optional `passive.max_tasks` or `passive.default_interval` later.

---

## Dependency Summary

| Feature        | New dependency (suggested)   | Platform   |
|----------------|------------------------------|------------|
| Monitor        | `github.com/vishvananda/netlink` | Linux only |
| Passive        | Same netlink or `/proc/net/dev`  | Linux (netlink or proc) |

Use of netlink for both features allows a single dependency and consistent behavior on Linux.

---

## Suggested Order of Implementation

1. **Feature 2 (Passive) first**  
   - Smaller surface: polling loop + existing task/result pattern; no event-driven netlink subscription.  
   - Delivers visible value (interface counters) and validates “another task type” in the same API/CLI.

2. **Feature 1 (Monitor) second**  
   - Introduces netlink listener and new “monitor” resource (separate from measurement tasks).  
   - Builds on same netlink dependency and clarifies event vs measurement semantics.

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Netlink listener blocks or panics | Run in dedicated goroutine; recover in handler; log and continue. |
| Route events too frequent | Phase 2 only; add filters (e.g. by table, prefix); optional rate limit per task. |
| Passive polling overhead | Limit concurrent passive tasks; use reasonable default interval (e.g. ≥ 5s). |
| Linux-only | Document clearly; optional build tags or stubs for non-Linux (no-op or “not supported”). |

---

## Summary

- **Monitor tasks**: Add a new “monitors” resource and netlink-based listener in lopad; report interface/IP (and later route) changes via webhook and optional event API; CLI `lopa monitor`.
- **Passive measurement**: Add “passive” task type that polls interface counters at an interval; reuse task/result API and CLI patterns; extend result schema for bytes/packets.

Both features fit the existing Lopa architecture (lopad + REST + CLI) and can be implemented in phases with the above scope and APIs.
