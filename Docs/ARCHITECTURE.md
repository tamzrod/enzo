# ARCHITECTURE.md

## ENZO Compression Engine Architecture

**State‑synchronized, agreement‑based stream transformer (TCP‑in / TCP‑out)**

---

## 1. Purpose and Scope

ENZO is a **transparent, inline stream transformation layer** designed to sit between two TCP endpoints.

Its primary goal is to **reduce bandwidth opportunistically** by exploiting **structural repetition** in streaming data through **end‑to‑end agreement**, not through entropy coding.

ENZO is:

* protocol‑agnostic
* connection‑scoped
* stateful but reset‑safe
* invisible to upstream and downstream applications

ENZO is **not**:

* a file compressor
* a message broker
* an application‑level protocol
* a replacement for TLS or transport security

---

## 2. Core Architectural Principles

These principles are **authoritative**. Any implementation or extension must preserve them.

### 2.1 Agreement, Not Guessing

Compression is achieved through **shared agreement state** between sender and receiver:

* dictionaries
* templates
* epochs

The sender defines meaning.
The receiver applies meaning.

The receiver **never guesses**.

---

### 2.2 Magic Is the Only Mandatory Header

A single **magic byte / marker** is the only unconditional signal required on the wire.

Magic answers the only always‑necessary question:

> *“What am I looking at?”*

Once magic is observed and state is established:

* protocol version is known
* mode is known
* interpretation rules are known

All other metadata is **conditional** and **state‑dependent**.

---

### 2.3 Silence Is Meaningful

The **absence of control output is itself meaningful**.

* When state does not change, no control information is required
* Repeating headers when nothing has changed is considered noise
* Stability is communicated by **silence**, not repetition

Control frames exist **only** to signal state transitions.

---

### 2.4 Headers Represent Control, Not Payload

Headers are **control traffic**, not data.

They exist only to signal:

* epoch resets
* dictionary or template definition
* interpretation changes

Payload data must **not continuously pay a header tax** when state is stable.

---

### 2.5 Opportunistic Compression

Compression is an **optimization**, never a requirement.

ENZO must:

* observe and learn continuously
* compress only when profitable
* transparently pass raw data when compression provides no gain

If compression cannot prove benefit, ENZO must **get out of the way**.

---

### 2.6 Raw Passthrough Is a Valid Steady State

Passing raw data unchanged is:

* correct behavior
* a valid long‑term operating mode
* not a failure condition

ENZO must be safe to deploy inline with **zero regret**.

---

### 2.7 State Over Repetition

Meaning is carried by **shared state**, not repeated metadata.

Once state is synchronized:

* data alone may be sufficient
* headers are omitted unless state changes

Epoch reset is the only hard resynchronization mechanism.

---

## 3. TCP‑In / TCP‑Out Model

ENZO relies entirely on TCP for:

* ordering
* reliability
* backpressure

ENZO does **not** re‑implement transport semantics.

It behaves as a pure stream transformer:

```
TCP → ENZO → TCP
```

---

## 4. One Connection = One Context

Each TCP connection owns exactly one:

* dictionary
* epoch timeline
* compression context

Contexts are **never shared** across connections.

All state is scoped to the lifetime of the TCP connection.

---

## 5. Operational Model

ENZO runs as a single service that:

* listens on one TCP port
* forwards traffic to one configured destination

Multiple simultaneous connections are supported.
Each connection is isolated and independent.

Routing decisions are made **per connection**, never globally.

---

## 6. Stream Classification (Encode vs Decode)

ENZO auto‑selects behavior by inspecting the incoming stream.

### Classification Rule

* **Magic byte present** → stream is ENZO‑encoded → **decode mode**
* **Magic byte absent** → stream is raw → **encode mode**

This decision is:

* deterministic
* content‑driven
* configuration‑free

Ports and configuration do not imply meaning.

---

## 7. Epoch Model

### 7.1 Definition

An **epoch** is a bounded period during which both sides share identical agreement state.

Dictionary IDs and template IDs are valid **only within their epoch**.

---

### 7.2 Reset Semantics

If either side:

* loses state
* restarts
* detects protocol violation

Then:

* a new epoch begins
* both sides discard dictionary state
* compression restarts naturally

The dictionary is treated as a **cache**, not durable state.
No replay is performed.

---

## 8. Dictionary Model

### 8.1 Nature

The dictionary is:

* bounded
* disposable
* per‑connection

Losing the dictionary is acceptable and safe.

---

### 8.2 Lifespan

Dictionary entries:

* live within an epoch
* age only when unused
* are refreshed on use

When bounds are exceeded, a new epoch is started.

---

## 9. Units of Compression

### 9.1 Templates

Templates capture **structural repetition**:

* constant byte segments
* variable lanes

Templates are defined once and referenced many times.

This works especially well for:

* telemetry
* line‑oriented protocols
* logs
* industrial and control traffic

---

### 9.2 RAW Fallback

When data is:

* too small
* too large
* insufficiently repetitive
* unsafe to model

ENZO emits RAW bytes unchanged.

Correctness always wins over compression.

---

## 10. Wire Protocol Philosophy

ENZO uses an **event‑based wire protocol**.

Key properties:

* frames represent **control events**, not steady‑state data
* headers are emitted **only on state transitions**
* absence of frames implies unchanged state

The protocol is hard‑defined, not negotiated.

See `PROTOCOL.md` for exact wire details.

---

## 11. Failure Model

### 11.1 Receiver Restart

If the receiver restarts:

* TCP connection drops
* state is lost
* a new epoch begins

Traffic resumes safely on reconnect.

---

### 11.2 Invalid Streams

On protocol violation:

* decoding stops
* epoch is reset or connection is closed

Silent corruption is never allowed.

---

## 12. Configuration Philosophy

### 12.1 Separation of Concerns

ENZO separates:

* **protocol invariants** (meaning)
* **operational policy** (limits and deployment)

---

### 12.2 Protocol Invariants (Not Configurable)

The following are fixed by architecture:

* magic semantics
* encode/decode routing
* epoch behavior
* dictionary agreement model
* strict decoding rules

Any attempt to configure these is ignored or rejected.

---

### 12.3 Operational Defaults (Overridable)

Operational limits may be overridden only when explicitly provided:

* connection limits
* dictionary caps
* idle timeouts
* logging verbosity

If omitted, defaults apply unchanged.

---

## 13. Minimal Configuration Surface (v1)

```yaml
listen:
  address: 0.0.0.0
  port: 9000

destination:
  address: 127.0.0.1
  port: 8086
```

All other behavior derives from architectural defaults.

---

## 14. Security Position

ENZO is **not a security protocol**.

* assumes trusted or externally secured transport
* may leak structural information
* does not replace TLS or VPNs

---

## 15. Future: Dictionary Encryption

Future versions may support:

* encryption of dictionary definitions only
* epoch‑scoped keys
* external key management

Encryption is optional and orthogonal to compression logic.

---

## 16. Summary

ENZO works because:

* agreement replaces repetition
* magic anchors interpretation
* silence communicates stability
* dictionaries are disposable
* epochs make recovery trivial
* compression is optional and honest

The system favors **correctness, minimal overhead, and operability** over theoretical optimality.
