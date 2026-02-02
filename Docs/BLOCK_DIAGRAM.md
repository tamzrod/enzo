# BLOCK_DIAGRAM.md

## Compression Engine – Block Diagram (TCP-in / TCP-out)

This document describes the **high-level block diagram** and data flow of the compression engine.

The goal is to make the **runtime topology obvious** before any code is written.

---

## 1. High-Level Concept

The engine sits **inline** between two TCP endpoints.

It behaves like a transparent pipe:

- Input: TCP stream
- Output: TCP stream
- Transformation: agreement-based compression (+ optional encryption)

```
   Upstream App / Device
            │
            │ TCP
            ▼
   ┌──────────────────┐
   │  Listening Port  │  (Ingress)
   └──────────────────┘
            │
            ▼
   ┌──────────────────┐
   │   Framer / I/O   │  (stream → records)
   └──────────────────┘
            │
            ▼
   ┌──────────────────┐
   │ Agreement Layer  │
   │ (Dictionary +    │
   │  Templates +     │
   │  Epochs)         │
   └──────────────────┘
            │
            ▼
   ┌──────────────────┐
   │   Event Encoder  │
   └──────────────────┘
            │
            ▼
   ┌──────────────────┐
   │ Optional Encrypt │  (future)
   │ (Dict defs only) │
   └──────────────────┘
            │
            ▼
   ┌──────────────────┐
   │   TCP Sender     │  (Egress)
   └──────────────────┘
            │
            │ TCP
            ▼
        Remote End
```

---

## 2. Sender-Side Blocks (Encode Mode)

### 2.1 Listening Port

- Accepts incoming TCP connections
- Acts as a **proxy ingress**
- No protocol awareness

Responsibility:
- Byte stream in
- Byte stream out

---

### 2.2 Framer / I/O Layer

Purpose:
- Convert raw TCP byte stream into **records** or **frames** suitable for analysis

Notes:
- For line-based protocols: delimiter-based framing
- For packetized sources: pass-through framing
- Framing is **pluggable** and outside core compression logic

---

### 2.3 Agreement Layer (Core Engine)

This is the **heart of the system**.

Responsibilities:
- Detect repeated structure
- Learn templates
- Manage dictionary lifecycle
- Enforce min/max bounds
- Manage epoch state

Key properties:
- Stateful
- Deterministic
- Bounded memory

This layer decides whether to emit:
- TEMPLATE_DEFINE
- TEMPLATE_REF
- RAW_DATA
- EPOCH_RESET

---

### 2.4 Event Encoder

Purpose:
- Serialize semantic events into wire frames

Responsibilities:
- Build protocol headers
- Enforce definition-before-reference
- Length-prefix frames

This layer has **no compression logic** — only serialization.

---

### 2.5 Optional Encryption Layer (Future)

Scope:
- Encrypt dictionary definitions only
- Does NOT encrypt RAW payloads
- Does NOT encrypt control frames

Placement:
- After event encoding
- Before TCP send

Reason:
- Keeps compression logic plaintext
- Keeps crypto orthogonal

---

### 2.6 TCP Sender

Purpose:
- Send framed events over TCP
- Rely on TCP for:
  - ordering
  - retransmission
  - flow control

---

## 3. Receiver-Side Blocks (Decode Mode)

Receiver mode runs the **same binary**, but with I/O direction reversed.

Instead of accepting application data and forwarding compressed data, it:
- accepts **compressed protocol frames**
- reconstructs the original byte stream
- forwards it to a destination TCP endpoint

```
Compressed TCP In
        │
        ▼
┌──────────────────┐
│   TCP Receiver   │  (Ingress)
└──────────────────┘
        │
        ▼
┌──────────────────┐
│ Optional Decrypt │  (future)
│ (Dict defs only) │
└──────────────────┘
        │
        ▼
┌──────────────────┐
│  Event Decoder   │
│ (Frame parsing)  │
└──────────────────┘
        │
        ▼
┌──────────────────┐
│ Agreement Layer  │
│ (Apply events,   │
│  manage epochs)  │
└──────────────────┘
        │
        ▼
┌──────────────────┐
│  Framer / I/O    │
│ (records → bytes)│
└──────────────────┘
        │
        ▼
┌──────────────────┐
│ Destination TCP  │  (Egress)
└──────────────────┘

### Responsibilities

- Strictly apply events in order
- Maintain dictionary and epoch state
- Never guess missing definitions
- Trigger epoch reset on protocol violation

---

## 4. Control vs Data Flow

Control and data are **multiplexed on the same TCP stream**:

- Control: EPOCH_RESET, DEFINE
- Data: REF, RAW

This keeps:
- ordering guaranteed
- synchronization trivial

---

## 5. Failure and Reset Flow

### Receiver Restart

```
Receiver crash
→ Receiver restarts (empty state)
→ Signals reset (or reconnect)
→ Sender emits EPOCH_RESET
→ Both sides clear dictionary
→ Compression restarts
```

No dictionary resend. No replay.

---

## 6. Why This Block Design Works

- TCP handles reliability
- Agreement layer removes redundancy
- Dictionary is disposable
- Epoch reset makes recovery trivial
- Encryption is optional and isolated

Each block has **one responsibility**.

---

## 7. Summary

This block diagram establishes:
- clear data flow
- clean separation of concerns
- correctness-first recovery
- future extensibility

With this diagram, implementation can proceed without architectural guesswork.

