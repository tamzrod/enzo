# ENZO Docker Deployment

ENZO is a protocol-agnostic TCP proxy for compression. This deployment provides the web UI for configuration and runs ENZO processes.

## Quick Start

```bash
cd deploy/docker
docker-compose up -d
```

Access the web UI at: http://localhost:8282

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Host Machine (host network mode)                          │
│                                                             │
│  ┌──────────────────┐                                       │
│  │     WebUI        │  Configure & Start ENZO               │
│  │    :8282         │                                       │
│  └────────┬─────────┘                                       │
│           │                                                 │
│           ▼                                                 │
│  ┌──────────────────┐     ┌──────────────────────────────┐│
│  │   ENZO Process    │────►│  External Services          ││
│  │   (spawned by    │     │  (Backend, Server ENZO, etc) ││
│  │    WebUI)        │     └──────────────────────────────┘│
│  └──────────────────┘                                       │
└─────────────────────────────────────────────────────────────┘
```

## Web UI Configuration

### Server Mode
- **Listen Port**: Port ENZO listens on (default: 8086)
- **Backend**: IP:Port of your backend service (e.g., InfluxDB)

### Client Mode
- **Listen Port**: Port for edge devices to connect to (default: 8088)
- **Server ENZO**: IP:Port of server-side ENZO

## Manual Docker Commands

```bash
# Build image
docker build -f deploy/docker/Dockerfile -t enzo .

# Run container (host network mode)
docker run -d --name enzo-webui \
  --network host \
  enzo

# Access web UI at http://localhost:8282
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| PORT | 8282 | Web UI port |

## Ports

The container uses host network mode to allow ENZO processes to:
- Listen on any port
- Connect to external services
