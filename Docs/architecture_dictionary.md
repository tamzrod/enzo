# ENZO Dictionary Architecture (Frozen Spec)

## Purpose
This document defines the **dictionary learning subsystem** for ENZO.
It is **authoritative and frozen** for v1.

The dictionary exists to opportunistically learn repeated byte patterns
*without ever interfering with RAW streaming*.

---

## Core Principles (Non‑Negotiable)

1. **RAW is sacred**
   - RAW bytes always flow immediately
   - Dictionary failure must never block RAW

2. **Packet‑local discovery**
   - Candidate patterns are discovered *only inside a single packet*
   - No cross‑packet stitching, ever

3. **Window‑bounded learning**
   - Evidence is valid only while inside the 2 MB RAW window
   - No timers, no decay math

4. **Conservative promotion**
   - Patterns must prove usefulness before entering the dictionary
   - Learning is delayed, not eager

---

## Definitions

### Packet
A packet is the atomic observation unit.
Examples:
- One HTTP request body
- One Influx write batch
- One decoder frame payload

Packets are assumed complete or ignored.

---

### Span (Candidate Pattern)

A span has the form:
```
[CONST_A][VAR][CONST_B]
```

Rules:
- Entire span must exist inside one packet
- VAR must be non‑empty
- Byte‑exact (no parsing, no semantics)

---

## Overlap Rules

- Overlap is allowed **inside a packet**
- Longest valid span has highest priority
- Shorter overlapping spans may exist but score lower

Overlap across packets occurs only via **repetition**, not concatenation.

---

## Hit Accounting

### Intra‑packet Hits
- Count of full span occurrences inside one packet
- Used as **strong local evidence**

### Inter‑packet Hits
- Count of packets where the full span appears
- Used as **stability evidence over time**

---

## Qualification Rules

A candidate qualifies for dictionary entry if **either** condition is met:

### Path A — Cross‑packet Stability
```
interPacketHits ≥ threshold
```

### Path B — Strong Single‑packet Evidence
```
intraPacketHits ≥ threshold
AND span_size ≥ MIN_SIZE
```

All evidence must fall within the active RAW window.

---

## IDs

- Type: `uint32`
- Assigned **only on qualification**
- Never recycled within an epoch

---

## Scoring

```
score = hits × effective_size
```

- hits includes intra + inter contributions
- effective_size approximates bytes saved

---

## Eviction Policy

1. **Age‑gated eligibility**
   - Only older half of entries may be evicted

2. **Score‑based selection**
   - Lowest score evicted first

If eviction cannot free enough space:
- Reject new candidate
- Continue RAW streaming normally

---

## Epoch Behavior

- Epoch ends only when ID space is exhausted
- Make‑before‑break transition
- New epoch starts before old is retired

---

## Non‑Goals (Explicitly Out of Scope)

- No protocol awareness
- No semantic parsing
- No adaptive tuning (yet)
- No control plane

---

## One‑Line Summary

> The ENZO dictionary is a window‑bounded, packet‑local, conservative learning cache where RAW always wins and patterns earn survival through repetition and usefulness.

---

**Status:** FROZEN

