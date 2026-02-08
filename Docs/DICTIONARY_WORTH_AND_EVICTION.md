# Dictionary Worth and Eviction (v1)

## Worth
Dictionary entries accumulate value based on hit count and size.

## Eviction
When capacity is reached:
1. Entries are split by age
2. Only the older half is eligible
3. Lowest-worth entries are evicted first

This behavior is locked for v1.
