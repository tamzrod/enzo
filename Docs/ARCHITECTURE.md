# ENZO Architecture

## Core Principles (LOCKED)

### 1. Packet Transformation (Primary Invariant)

**ENZO is a packet transformer.**

> **One payload in → one ENZO frame out.**
>
> Any internal re-segmentation (line-based, delimiter-based, heuristic-based) is a **hard violation**.

Compression that breaks packet boundaries does not save bandwidth; it **multiplies overhead** and destroys ROI.

This rule is non-negotiable.

---

### 2. Explicitness (LOCKED)

> **If ENZO touches the data, it MUST emit an ENZO frame.**  
> **Everything ENZO outputs begins with the ENZO magic byte.**

There is no invisible passthrough.

Even when compression is skipped, the decision is made explicit via an ENZO frame.

This guarantees:
- deterministic behavior
- safe chaining
- bounded worst-case loss
- no re-encoding ambiguity

---

## What ENZO Is

ENZO is a **payload-oriented agreement engine** that sits between systems and transforms packets while preserving exact byte boundaries.

ENZO:
- operates on **whole payloads**, not records or lines
- treats payload bytes as **opaque**
- never invents structure
- never infers boundaries
- never parses application protocols

ENZO applies agreement **only within the payload it receives**.

---

## What ENZO Is Not

ENZO is **not**:
- a line protocol parser
- a stream parser
- a record splitter
- a text-aware compressor
- a delimiter-driven engine
- a transparent proxy

Any logic based on `\n`, `\r`, textual structure, or implicit stream semantics is forbidden in the ENZO core.

---

## Mode Detection (LOCKED)

ENZO determines its operating mode **once per connection**, using explicit on-wire identity:

- If the first byte equals the ENZO magic byte → **DECODE mode**
- Otherwise → **ENCODE mode**

This decision:
- is content-driven
- requires no configuration
- never changes during the lifetime of the connection

---

## Packet Semantics

### Encoder

- Reads **exactly one payload chunk** from the upstream producer
- Applies agreement logic **only within that payload**
- Emits **exactly one ENZO frame**

If agreement produces negative ROI:
- ENZO emits a **RAW_DATA frame**
- The payload is transmitted unchanged
- The ENZO header is still present

There is a strict 1:1 correspondence between input payloads and output frames.

---

### Decoder

- Reads **exactly one ENZO frame**
- Reconstructs the payload deterministically
- Emits **exactly one payload chunk** downstream

Decoder output must be **byte-for-byte identical** to encoder input for the same payload.

---

## RAW Semantics (CLARIFIED)

**RAW does not mean passthrough.**

RAW means:
- ENZO is active
- a compression attempt was evaluated
- compression was intentionally skipped
- the decision is explicitly framed

RAW frames:
- always include the ENZO magic byte
- never affect agreement state
- guarantee symmetry and idempotence

---

## Adapter Responsibility

Boundary decisions belong to adapters, not ENZO.

Producers (gateways, agents, test harnesses):
- decide what constitutes a complete dataset
- assemble full payloads
- emit payloads atomically

ENZO:
- receives payloads
- transforms them
- preserves boundaries exactly

---

## Agreement Scope

Agreement is:
- **payload-scoped**
- **stateful across packets**
- **never inferred**

If agreement cannot amortize header cost **within a single payload**, ENZO must fall back to RAW.

Negative ROI is allowed but **explicitly framed**.

---

## Cost Model (LOCKED)

- The **worst-case loss** for ENZO is the ENZO header size
- This loss is:
  - fixed
  - bounded
  - paid at most once per payload

There is:
- no cascading overhead
- no hop-dependent amplification
- no hidden expansion

This bounded loss is an intentional design trade.

---

## Consequences of This Architecture

- Payload boundaries are preserved end-to-end
- Compression results are honest and measurable
- Chaining ENZO instances is always safe
- Transport quirks cannot sabotage correctness
- Worst-case behavior is predictable

Any regression toward invisible behavior, stream semantics, or implicit decisions violates this architecture.

---

## Final Summary (DO NOT REMOVE)

> **ENZO exists to transform packets, not streams.**
>
> **If ENZO touches the data, it leaves an explicit mark.**
>
> **The worst-case loss is the header — and nothing more.**