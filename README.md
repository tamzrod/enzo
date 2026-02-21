# ZENO

Zeno is a schema-driven configuration instrument.

It is designed with one core philosophy:

> Make it hard for the user to make a mistake.

Zeno is not a text editor. Zeno is not a YAML playground. Zeno is a
controlled environment for building valid configurations.

------------------------------------------------------------------------

## Why Zeno Exists

Modern systems rely heavily on structured configuration files.

These files are:

-   Powerful
-   Flexible
-   Easy to misuse
-   Easy to break

AI can generate configuration. But AI does not enforce correctness.

Zeno enforces correctness.

It provides deterministic validation, structural authority, and explicit
lifecycle control.

------------------------------------------------------------------------

## Core Principles

-   Schema is authoritative.
-   Core engine is UI-agnostic.
-   No live auto-generation.
-   Explicit Write and Generate phases.
-   Validation blocks invalid state transitions.
-   No popup validation errors.
-   Errors are highlighted, not hidden.

------------------------------------------------------------------------

## Architecture Overview

Zeno consists of:

-   Core (IR + Schema + Validator + Exporter)
-   Desktop UI (PySide6)
-   Future Web UI (separate adapter)

The core contains no UI logic.

Both desktop and web interfaces interact with the same deterministic
engine.

------------------------------------------------------------------------

## Operating Modes

Zeno operates in two modes:

### Schema Mode

Edit `.zs` schema definitions.

### Config Mode

Edit configuration constrained by a loaded schema.

A configuration cannot exist without a schema.

------------------------------------------------------------------------

## Lifecycle

1.  Load Schema
2.  Load Config
3.  Edit
4.  Write (validate structure + uniqueness)
5.  Generate (validate runtime constraints)
6.  Export

Mutation and projection are separated intentionally.

------------------------------------------------------------------------

## Validation Model

Two validation phases:

### Write Phase

-   Structural validation
-   Required fields
-   Type correctness
-   Array uniqueness (`unique_by`)

### Generate Phase

-   Structural re-check
-   Semantic rules
-   Cross-field constraints
-   Runtime export validation

Invalid state is blocked at the boundary.

------------------------------------------------------------------------

## Error Model

Errors are structured objects:

-   path
-   message
-   phase
-   severity

UI highlights the exact location of failure.

No modal popups. No silent corrections.

------------------------------------------------------------------------

## Current Scope

Zeno is currently focused on:

-   MMA configuration modeling
-   Strict nested model
-   Deterministic validation

Future expansion is possible, but generalization will only occur after
domain stability.

------------------------------------------------------------------------

## Status

Zeno is under active architectural development.

Documentation-first. Implementation second. No architectural drift.

------------------------------------------------------------------------

Zeno is not built for speed of typing.

It is built for correctness.
