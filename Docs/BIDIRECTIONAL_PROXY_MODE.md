# ENZO Bidirectional Proxy Mode (v1)

## Overview
ENZO operates as a transparent TCP proxy that can optimize traffic in both directions of a connection.

Bidirectional mode is required for response-heavy protocols such as:
- Modbus (read registers)
- Grafana → InfluxDB queries
- HTTP APIs

Each direction maintains its own learning state and dictionary.

## Architecture
Client → ENZO → Server
Client ← ENZO ← Server

Both lanes are optimized independently.
