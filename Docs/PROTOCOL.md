# PROTOCOL.md

## ENZO Wire Protocol Specification

**Deterministic, event-based protocol for agreement-driven stream compression**

This document is aligned with `ARCHITECTURE.md` and defines the **exact on-the-wire meaning** of ENZO traffic.

The protocol is:
- explicit
- framed
- versioned
- non-negotiable

It exists to define **meaning**, not configuration.

---

## 1. Protocol Scope

The ENZO protocol defines:
- how agreement events are encoded on the wire
- how epochs are reset
- how templates are defined and referenced
- how raw data is transported safely

The protocol does **not**:
- negotiate features
- expose runtime knobs
- encode transport reliability
- implement encryption (except future dictionary encryption)

---

## 2. Stream Identity and Routing

ENZO streams are **self-identifying**.

### Routing Rule (Invariant)

- If the first byte equals the ENZO magic byte → the stream is **compressed** and must be **decoded**
- Otherwise → the stream is **raw** and must be **encoded**

This decision is:
- content-driven
- per-connection
- independent of ports or configuration

---

## 3. Framing Model

The TCP byte stream is segmented into **frames**.

Each frame:
- represents exactly one semantic event
- is length-prefixed
- is processed strictly in order

Frames are never nested.

---

## 4. Frame Header (Fixed, v1)

All frames begin with the same fixed-size header:

```
+--------+--------+--------+--------+
| Magic  | Version| Type   | Flags  |
+--------+--------+--------+--------+
|        Length (uint32 BE)          |
+-----------------------------------+
```

### Header Fields

| Field   | Size | Meaning |
|-------|------|--------|
| Magic | 1 B | Fixed value `0xEC` identifying ENZO protocol |
| Version | 1 B | Protocol version (v1 = `0x01`) |
| Type | 1 B | Frame type (event kind) |
| Flags | 1 B | Reserved (must be `0x00` in v1) |
| Length | 4 B | Payload length in bytes |

Total header size: **8 bytes**

If `Magic` does not match, the stream must not be interpreted as ENZO protocol.

---

## 5. Frame Types (v1)

| Type | Name | Description |
|----|------|-------------|
| `0x01` | EPOCH_RESET | Reset dictionary and epoch state |
| `0x02` | TEMPLATE_DEFINE | Define a new template |
| `0x03` | TEMPLATE_REF | Reference an existing template |
| `0x04` | RAW_DATA | Literal byte payload |

All other type values are reserved.

---

## 6. Epoch Semantics

### 6.1 EPOCH_RESET Frame

An epoch reset synchronizes state between sender and receiver.

#### Payload

```
+-------------------+
| Epoch ID (uint32) |
+-------------------+
```

Rules:
- receiver must discard all dictionary state
- sender must stop referencing prior IDs
- epoch ID is monotonically increasing (wrap allowed)

Epoch reset is the **only recovery mechanism** in v1.

---

## 7. Template Definition

### 7.1 TEMPLATE_DEFINE Frame

Defines a reusable structural template.

#### Payload

```
+-------------------+
| Template ID (u16) |
+-------------------+
| Segment Count (u8)|
+-------------------+
| Segment Data ...  |
```

Each segment:

```
+-----------+--------------------+
| Seg Type  | Segment Length (u16)|
+-----------+--------------------+
| Segment Bytes / Metadata       |
+--------------------------------+
```

Segment Types (v1):
- `0x01` CONST_BYTES
- `0x02` VAR_LANE

Templates **must be fully defined before use**.

---

## 8. Template Reference

### 8.1 TEMPLATE_REF Frame

References a previously defined template and supplies lane values.

#### Payload

```
+-------------------+
| Template ID (u16) |
+-------------------+
| Lane Count (u8)   |
+-------------------+
| Lane Payloads ... |
```

Each lane payload:

```
+--------------------+
| Lane Length (u16)  |
+--------------------+
| Lane Bytes         |
+--------------------+
```

Reconstruction order is strictly defined by template layout.

---

## 9. RAW_DATA Frame

RAW frames carry literal bytes unchanged.

```
+--------------------+
| Raw Bytes ...      |
+--------------------+
```

RAW_DATA may appear at any time and never affects dictionary state.

---

## 10. Ordering and Validity Rules

The following invariants are mandatory:

1. Frames are processed strictly in order
2. TEMPLATE_DEFINE must precede TEMPLATE_REF
3. References to unknown IDs are protocol violations
4. Frames from a previous epoch are invalid

On violation:
- decoding stops
- epoch reset or connection termination occurs

Silent recovery is forbidden.

---

## 11. Error Handling Model

ENZO favors **fail-fast correctness**:

- invalid headers → hard fail
- unknown frame type → skip via length
- invalid payload structure → reset or close

No guessing is allowed.

---

## 12. Forward Compatibility

- Unknown frame types may be skipped using `Length`
- Unknown flags must be ignored
- Version mismatch must fail hard

Protocol evolution requires a new version number.

---

## 13. Relationship to Configuration

The protocol is **not configurable**.

- magic byte
- frame layout
- semantics
- invariants

are fixed by architecture.

Configuration only affects **operational limits**, not protocol meaning.

---

## 14. Future: Dictionary Encryption

Future versions may encrypt:
- template definitions

Encryption properties:
- epoch-scoped
- definition-only
- external key management

Decryption failure triggers epoch reset.

---

## 15. Summary

The ENZO protocol:

- defines meaning explicitly
- enforces agreement deterministically
- resets safely via epochs
- never guesses
- never negotiates

It is designed to be **boring, strict, and reliable**.

