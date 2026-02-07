# ENZO – RAW Window & Dictionary Eviction Concept (Locked)

> **Purpose:** Preserve the *concept* even if the code is lost.  
> This document captures the reasoning, invariants, and eviction rules behind ENZO’s RAW and Dictionary memory design.

---

## 1. Design Philosophy (Core Principles)

ENZO is a **streaming, full‑duplex proxy**.

- **RAW stream is the truth** – data must flow immediately
- **Dictionary is optional intelligence** – learning must never block traffic
- **Memory is strictly bounded** – behavior must be predictable
- **Failure is cheap** – RAW fallback is always allowed

> If compression fails, ENZO still works.

---

## 2. RAW Memory Window (2 MB)

### What it is
- A **sliding FIFO observation window**
- **Hard‑capped at 2 MB**
- Used only for **pattern observation**, not buffering

### What it is NOT
- ❌ Not a queue
- ❌ Not a batch buffer
- ❌ Not a delay mechanism

### Behavior
- RAW bytes are **forwarded immediately** to the wire
- Simultaneously, bytes are appended to the RAW window
- If window exceeds 2 MB:
  - Evict **oldest bytes first (FIFO)**
  - Window size remains constant

> **Invariant:** RAW overflow must never affect wire flow.

---

## 3. Dictionary Memory (1 MB)

### Purpose
- Store **reusable byte patterns** discovered from RAW stream
- Enable future reference‑based encoding (later phase)

### Constraints
- **Hard cap:** 1 MB total
- Dictionary entries are:
  - Opportunistic
  - Disposable
  - Never required for correctness

> RAW always wins over Dictionary.

---

## 4. Dictionary Entry Lifecycle

Each dictionary entry has:
- Size (bytes)
- Creation age
- Last‑used timestamp (for LRU)

Entries are admitted only if there is space or evictable space.

---

## 5. Eviction Policy (Locked v1)

### High‑level rule

> **Evict by LRU, but only from the older half of the dictionary.**

This is a **guarded LRU** strategy.

---

### Step‑by‑step eviction logic

When inserting a new dictionary candidate:

1. Check free space
2. If enough → insert
3. If not enough:
   - Sort dictionary entries by **age**
   - Split into two halves:
     - **Young half (newer 50%)** – *protected*
     - **Old half (older 50%)** – *evictable*
   - From the **old half only**:
     - Evict entries using **Least Recently Used (LRU)**
     - Continue until enough space is freed
4. If still not enough space:
   - **Reject the new candidate**
   - Continue streaming RAW normally

---

## 6. Why This Policy Exists

### Problems avoided
- Premature eviction of new patterns
- Dictionary thrashing during burst traffic
- High‑ROI entries being evicted due to timing
- Hidden coupling between compression and streaming

### What this guarantees
- New entries get a **fair chance** to prove value
- Frequently used entries are naturally protected
- Dictionary behavior is **predictable and explainable**

---

## 7. Mental Model (Important)

- **RAW window** → audition stage
- **Dictionary** → probationary cache
- **Older half eviction** → performance review

> No one gets evicted on day one.

---

## 8. Non‑Goals (Explicit)

This design intentionally avoids:
- Adaptive tuning
- Dynamic resizing
- ROI scoring formulas
- Compression‑driven backpressure
- Learning that can break streaming

These may come later, but are **out of scope** for v1.

---

## 9. One‑Line Summary (If You Forget Everything Else)

> **RAW always flows; the dictionary only learns when it can afford to.**

---

## 10. Status

- Concept: **LOCKED**
- Memory sizes: **Hard‑coded**
- Behavior: **Deterministic, bounded, streaming‑safe**

This document is the source of truth for the idea, even if the implementation changes.

