# Reverse Fast-Path (TTFB Protection) — v1 Locked

## Purpose
Small responses such as HTTP status replies are latency-sensitive and not worth optimizing.

## Rule
Reverse traffic is passed through directly until 256 bytes are observed.
After crossing this threshold, ENZO optimization is enabled permanently for the connection.

This preserves time-to-first-byte while still optimizing large responses.
