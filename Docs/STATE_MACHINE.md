# STATE_MACHINE.md

## ENZO State Machines

This document defines the **explicit state machines** for ENZO at runtime.

State machines are defined **per TCP connection**.
There is no global protocol state.

The purpose of this document is to:
- eliminate implicit behavior
- make failure handling explicit
- guarantee deterministic recovery

---

## 1. Core Invariants

These invariants apply to **all states**:

- One TCP connection = one compression context
- Dictionary and epoch are scoped to the connection
- Sender defines meaning; receiver applies meaning
- Receiver never guesses
- Protocol violations never silently recover

---

## 2. High-Level Connection Lifecycle

```
[ACCEPT]
   ↓
[CLASSIFY STREAM]
   ↓
[ENCODE MODE] or [DECODE MODE]
   ↓
[ACTIVE]
   ↓
[RESET or CLOSE]
```

---

## 3. Stream Classification State

### STATE: CLASSIFY_STREAM

**Entry:** TCP connection accepted

**Action:**
- Peek first byte (do not consume)

**Transitions:**
- If byte == MAGIC → `DECODE_INIT`
- Else → `ENCODE_INIT`

This decision is final for the lifetime of the connection.

---

## 4. Encode Path (Raw → Compressed)

### STATE: ENCODE_INIT

**Entry Actions:**
- Initialize dictionary
- Set epoch = 0
- Initialize framer

**Transition:**
- Immediately to `ENCODE_ACTIVE`

---

### STATE: ENCODE_ACTIVE

**Responsibilities:**
- Read raw bytes from source
- Frame input into records
- Detect structure
- Emit protocol frames

**Emitted Events:**
- TEMPLATE_DEFINE
- TEMPLATE_REF
- RAW_DATA
- EPOCH_RESET

**Transitions:**
- On dictionary limit exceeded → `ENCODE_EPOCH_RESET`
- On upstream close → `CLOSE`
- On fatal error → `CLOSE`

---

### STATE: ENCODE_EPOCH_RESET

**Entry Actions:**
- Increment epoch
- Emit `EPOCH_RESET`
- Clear dictionary

**Transition:**
- Return to `ENCODE_ACTIVE`

---

## 5. Decode Path (Compressed → Raw)

### STATE: DECODE_INIT

**Entry Actions:**
- Initialize empty dictionary
- Set epoch = unknown
- Initialize event decoder

**Transition:**
- To `DECODE_ACTIVE`

---

### STATE: DECODE_ACTIVE

**Responsibilities:**
- Read protocol frames
- Validate headers
- Apply events strictly
- Reconstruct raw byte stream

**Valid Incoming Frames:**
- EPOCH_RESET
- TEMPLATE_DEFINE
- TEMPLATE_REF
- RAW_DATA

**Transitions:**
- On EPOCH_RESET → `DECODE_RESET`
- On protocol violation → `DECODE_ERROR`
- On upstream close → `CLOSE`

---

### STATE: DECODE_RESET

**Entry Actions:**
- Clear dictionary
- Update epoch

**Transition:**
- Return to `DECODE_ACTIVE`

---

### STATE: DECODE_ERROR

**Entry Actions:**
- Stop decoding
- Signal error

**Transition:**
- Either:
  - `CLOSE`
  - or emit local reset and continue (implementation choice)

Silent recovery is forbidden.

---

## 6. Common Terminal State

### STATE: CLOSE

**Entry Actions:**
- Close source TCP connection
- Close destination TCP connection
- Release all resources

This state is terminal.

---

## 7. Failure Scenarios (Explicit)

### 7.1 Receiver Restart

- TCP connection drops
- State machine terminates
- New connection starts from `CLASSIFY_STREAM`
- New epoch begins

No state is reused.

---

### 7.2 Protocol Desynchronization

- Detected as invalid frame or unknown ID
- Transition to `DECODE_ERROR`
- Connection closed or reset

Correctness is preserved.

---

## 8. State Transition Summary

```
CLASSIFY_STREAM
  ├──> ENCODE_INIT ──> ENCODE_ACTIVE ──> ENCODE_EPOCH_RESET ──┐
  │                                                           │
  │                                                           └──> CLOSE
  └──> DECODE_INIT ──> DECODE_ACTIVE ──> DECODE_RESET ────────┐
                                                               │
                                                               └──> CLOSE
```

---

## 9. Why This State Machine Is Correct

- No hidden transitions
- All resets are explicit
- All failures are observable
- No shared mutable state
- Deterministic behavior per connection

---

## 10. Summary

The ENZO state machines:

- are simple by design
- map 1:1 to protocol invariants
- enforce agreement semantics
- make recovery trivial

With these states defined, implementation becomes mechanical rather than conceptual.

