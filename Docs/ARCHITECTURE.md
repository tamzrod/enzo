# ARCHITECTURE.md

## ENZO Compression Engine Architecture

**State‑synchronized, agreement‑based stream transformer (TCP‑in / TCP‑out)**

---

## 1. Purpose and Scope

ENZO is a **transparent, inline compression layer** designed to sit between two TCP endpoints.

It reduces bandwidth by exploiting **structural repetition** in streaming protocols through **end‑to‑end agreement**, not through traditional entropy coding.

ENZO is:
- protocol‑agnostic
- connection‑scoped
- stateful but reset‑safe
- invisible to upstream and downstream applications

It is **not** a file compressor, message broker, or application protocol.

---

## 2. Core Principles

### 2.1 Agreement, Not Guessing

Compression is achieved by both ends maintaining **shared agreement state** (dictionary + templates).

The sender *defines meaning*.
The receiver *applies meaning*.

The receiver never guesses.

---

### 2.2 TCP‑in / TCP‑out

ENZO relies entirely on TCP for:
- ordering
- reliability
- backpressure

ENZO does **not** re‑implement transport semantics.

---

### 2.3 One Connection = One Context

Each TCP connection owns exactly one:
- dictionary
- epoch timeline
- compression context

Contexts are **never shared** across connections.

---

## 3. Operational Model

ENZO runs as a **single service** that:
- listens on one TCP port
- forwards traffic to one configured destination

Multiple simultaneous connections are allowed.
Each connection is fully isolated.

Routing decisions are made **per connection**, not per process.

---

## 4. Stream Classification (Encode vs Decode)

ENZO auto‑selects behavior by inspecting the incoming stream.

### Rule

- **Magic byte present** → stream is compressed → **decode**
- **Magic byte absent** → stream is raw → **encode**

This decision is:
- deterministic
- content‑driven
- configuration‑free

Ports never imply meaning.

---

## 5. Epochs

### 5.1 Definition

An **epoch** is a time window during which both sides share the same dictionary state.

Dictionary IDs and template IDs are valid **only within their epoch**.

---

### 5.2 Reset Semantics

If either side loses state or detects protocol violation:

- a new epoch begins
- both sides discard dictionary state
- compression restarts naturally

No dictionary replay is performed.

The dictionary is treated as a **cache**, not durable state.

---

## 6. Dictionary Model

### 6.1 Nature

The dictionary is:
- bounded
- disposable
- per‑connection

Losing the dictionary is acceptable and safe.

---

### 6.2 Lifespan

Dictionary entries:
- live within an epoch
- age only when unused
- are refreshed on use

When bounds are exceeded, a new epoch is started.

---

## 7. Units of Compression

### 7.1 Templates

Templates capture **structural repetition**:
- constant byte segments
- variable lanes

Templates are defined once and referenced many times.

This works especially well for:
- line protocol
- telemetry
- logs
- industrial protocols

---

### 7.2 RAW Fallback

When data is:
- too small
- too large
- unsafe to model

ENZO emits RAW bytes unchanged.

Correctness always wins over compression.

---

## 8. Wire Protocol

ENZO uses an **explicit, framed, event‑based wire protocol**.

The protocol defines:
- frame headers
- event types
- ordering rules
- reset behavior

The protocol is **hard‑defined**, not negotiated.

See `PROTOCOL.md` for details.

---

## 9. Failure Model

### 9.1 Receiver Restart

If the receiver restarts:
- state is lost
- a new epoch begins
- both sides realign automatically

Traffic continues safely.

---

### 9.2 Invalid Streams

On protocol violation:
- decoding stops
- epoch is reset or connection is closed

Silent corruption is never allowed.

---

## 10. Configuration Philosophy

### 10.1 Separation of Concerns

ENZO distinguishes between:

- **Protocol invariants** (meaning)
- **Operational policy** (deployment limits)

---

### 10.2 Protocol Invariants (Not Configurable)

The following are fixed by architecture and never configurable:

- magic byte semantics
- encode/decode routing logic
- wire format
- definition‑before‑reference rule
- epoch semantics
- dictionary agreement model
- strict decoding behavior

Any attempt to configure these is ignored or rejected.

---

### 10.3 Operational Defaults (Overridable if Present)

The system has **sane defaults** for operational limits.

Configuration may override them **only when explicitly provided**:

- connection limits
- dictionary size caps
- idle timeouts
- logging verbosity

If omitted, defaults apply unchanged.

---

## 11. Configuration Surface (v1)

The minimal valid configuration is:

```yaml
listen:
  address: 0.0.0.0
  port: 9000

destination:
  address: 127.0.0.1
  port: 8086
```

All other behavior derives from architecture defaults.

---

## 12. Security Position

ENZO is **not a security protocol**.

- assumes trusted or externally secured transport
- may leak structural information
- does not replace TLS or VPNs

---

## 13. Future: Dictionary Encryption

Future versions may support:
- encryption of dictionary definitions only
- epoch‑scoped keys
- external key management

Encryption is optional and orthogonal to compression logic.

---

## 14. Summary

ENZO works because:

- agreement replaces repetition
- TCP guarantees correctness
- dictionaries are disposable
- epochs make recovery trivial
- configuration never alters meaning

The system favors **correctness, simplicity, and operability** over theoretical optimality.

