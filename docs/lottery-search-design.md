# Lottery Search System Design

## 1. Purpose and scope

This document defines a production-oriented design for searching and safely allocating lottery tickets from a dataset of 10 million or more records. It is a design-only deliverable. No lottery source code, database migration, or executable API is included.

The baseline design uses PostgreSQL as the system of record. Optional accelerators are separated from the correctness path and should be introduced only after measurement.

## 2. Requirements

### 2.1 Stated requirements

| Area | Requirement |
| --- | --- |
| Data volume | Support at least 10,000,000 ticket records. |
| Ticket number | Each ticket has a six-digit number. |
| Search pattern | Accept exactly six characters, each either `0`-`9` or `*`. |
| Wildcard semantics | A digit matches the same position; `*` matches any one digit. |
| Distribution | Concurrent requests using the same pattern must not receive the same ticket at the same time. |
| Performance | Search and allocation must be practical for 10M+ records. |
| Production design | Recommend storage, algorithm, indexing, concurrency control, and operational tradeoffs. |
| Deliverable | Produce a no-code design rather than a lottery implementation. |

Required examples:

| Pattern | Meaning |
| --- | --- |
| `****23` | Numbers ending in `23` |
| `1****5` | Numbers starting with `1` and ending in `5` |
| `123***` | Numbers starting with `123` |

### 2.2 Cardinality ambiguity

A six-digit decimal value has only:

```text
10^6 = 1,000,000 distinct values (000000 through 999999)
```

Ten million records cannot all have globally unique six-digit numbers. This design therefore separates a ticket instance from its searchable number:

- `ticket_id` uniquely identifies one ticket instance.
- `number` is the six-digit searchable value and may be duplicated.

Two tickets may both have number `123456`, but different `ticket_id` values make them independently allocatable. The same `ticket_id` must never be concurrently allocated twice.

If the interviewer intended globally unique numbers, one constraint must change: reduce the record count, increase the number width, or add a uniqueness dimension such as draw, batch, issuer, or series.

### 2.3 Explicit assumptions

These are working assumptions, not requirements stated by the prompt:

1. Each record represents an independently allocatable ticket instance.
2. `number` is treated as a fixed-width six-character value so leading zeroes are preserved.
3. `*` matches exactly one digit and matching covers all six positions.
4. Exclusivity is global per `ticket_id`, including requests with different but overlapping patterns.
5. Allocation begins as a temporary reservation and may later be consumed permanently.
6. Reservation duration is configurable business data; this document does not invent a fixed duration.
7. One allocation request reserves one ticket. Batch allocation requires separately defined all-or-nothing or partial-success semantics.
8. Database time is authoritative for reservation and expiry decisions.
9. No latency, QPS, availability, or retention target is assumed because none is supplied.

Questions requiring product clarification include reservation duration, batch size, whether partial fulfillment is allowed, audit retention, regional requirements, and whether number-level rather than instance-level exclusivity is intended.

## 3. Baseline architecture

```text
Clients
   |
   v
Load balancer
   |
   +----------------------+----------------------+
   v                      v                      v
API instance A       API instance B       API instance N
   |                      |                      |
   +----------------------+----------------------+
                          |
                          v
              PostgreSQL primary
              - ticket source of truth
              - indexes and query planning
              - row locks and transactions
              - reservations and audit events
                          |
               +----------+----------+
               v                     v
        read replica(s)         backups/monitoring
        analytics only          metrics/alerts

Optional after benchmarks:
API instances ---> Redis bitmap/cache ---> candidate ticket IDs
       |                                      |
       +---------- final validation and atomic claim in PostgreSQL
```

### 3.1 Allocation data flow

1. The API validates and parses the six-character pattern in constant time.
2. Fixed digits become database predicates; wildcard positions add no predicate.
3. A PostgreSQL transaction selects one eligible matching ticket with a row lock and `SKIP LOCKED`.
4. The same transaction changes the ticket to `RESERVED`, records its owner and expiry, and writes an audit event.
5. The transaction commits before the API returns the reservation.
6. Consume, release, and retry operations identify the reservation using `reservation_id` and an optional idempotency key.

API instances remain stateless. Correctness does not depend on a process-local mutex, cache, or sticky session.

## 4. Logical data model

### 4.1 Ticket

| Field | Logical type | Required | Purpose |
| --- | --- | --- | --- |
| `ticket_id` | Unique identifier | Yes | Immutable identity of one ticket instance |
| `number` | Six-character string | Yes | Canonical searchable value; not unique |
| `digit_1` ... `digit_6` | Single digit | Derived | Indexable positional representation of `number` |
| `state` | Enum | Yes | `AVAILABLE`, `RESERVED`, or `CONSUMED` |
| `reservation_id` | Unique identifier | While reserved | Identifies the current reservation attempt |
| `reserved_by` | Requester identifier | While reserved | Current reservation owner |
| `reserved_until` | UTC timestamp | While reserved | Reservation lease expiry |
| `consumed_reservation_id` | Unique identifier | When consumed | Reservation that completed consumption |
| `consumed_by` | Requester identifier | When consumed | Final allocation owner |
| `consumed_at` | UTC timestamp | When consumed | Final allocation time |
| `created_at` | UTC timestamp | Yes | Immutable creation time |
| `updated_at` | UTC timestamp | Yes | Latest state change time |
| `version` | Increasing integer | Yes | Optional compare-and-set concurrency guard |

`number` is the source of truth for the searchable value. The six digit-position fields are derived helpers and must be generated or validated so they cannot disagree with `number`.

### 4.2 TicketEvent

An append-only event provides auditability without becoming the current availability authority.

| Field | Purpose |
| --- | --- |
| `event_id` | Unique event identity |
| `ticket_id` | Affected ticket instance |
| `event_type` | `CREATED`, `RESERVED`, `RELEASED`, `RESERVATION_EXPIRED`, or `CONSUMED` |
| `reservation_id` | Correlation with a reservation attempt |
| `actor_id` | Requester or system actor |
| `from_state`, `to_state` | State transition |
| `occurred_at` | UTC event time |

The ticket state and related reservation/consumption fields are the current source of truth. Events explain how that state was reached. When complete auditing is required, the event and ticket mutation commit in the same database transaction.

### 4.3 IdempotencyRecord

Production clients may retry after timeouts. An idempotency record prevents a retry from allocating a second ticket.

| Field | Purpose |
| --- | --- |
| `requester_id`, `idempotency_key`, `operation` | Unique request identity |
| `request_hash` | Detects reuse of a key with different input |
| `ticket_id`, `reservation_id` | Result of the original operation |
| `response_status` | Stable completed result |
| `created_at`, `expires_at` | Retention window |

The idempotency record is created in the same transaction as allocation or consumption. A duplicate request with the same key and input returns the stored result; the same key with different input is rejected.

### 4.4 Data integrity constraints

1. `ticket_id` is globally unique and immutable.
2. `number` contains exactly six digits and is deliberately not unique.
3. Each digit helper equals its corresponding character in `number`.
4. `state` contains only `AVAILABLE`, `RESERVED`, or `CONSUMED`.
5. `AVAILABLE` has no reservation or consumption values.
6. `RESERVED` has `reservation_id`, `reserved_by`, and `reserved_until`, with no consumption values.
7. `CONSUMED` has `consumed_reservation_id`, `consumed_by`, and `consumed_at`, with no active reservation values.
8. Lifecycle timestamps use database UTC time; `updated_at` is not earlier than `created_at`.
9. `version` increases on each state mutation.
10. Consume or release must match both `ticket_id` and the current `reservation_id`.
11. Idempotency identity is unique per requester and operation.
12. Audit events are immutable and reference an existing ticket.

## 5. Ticket and reservation lifecycle

```text
AVAILABLE --reserve--> RESERVED --consume--> CONSUMED
    ^                       |
    +------ release --------+

expired RESERVED --atomic reclaim--> RESERVED with new reservation
expired RESERVED --maintenance------> AVAILABLE
```

### AVAILABLE

The ticket may be searched and claimed. Reservation and consumption fields are empty.

### RESERVED

The ticket is held exclusively by `reserved_by` until `reserved_until`. Only the owner presenting the current `reservation_id` may consume or release it.

An expired reservation is eligible again even if a maintenance worker has not changed its physical state back to `AVAILABLE`. Allocation may atomically replace an expired reservation. Cleanup is operational hygiene, not a correctness dependency.

### CONSUMED

The ticket is permanently allocated and excluded from all future candidate searches. `CONSUMED` is terminal unless a separately authorized and audited business process is introduced.

### Idempotent consume and safe retry

- The first valid consume locks the ticket, verifies owner, reservation ID, and non-expiry, then changes it to `CONSUMED`.
- A retry with the same idempotency key returns the original result.
- A retry with the same consumed reservation ID may also return the already-consumed result.
- A stale or expired reservation cannot consume a ticket that has since been re-reserved.
- A consume racing with expiry/reallocation is serialized by the ticket row lock; only the first valid transaction succeeds.

The configured reservation TTL is applied using database time, reducing clock-skew risk between API instances.

## 6. Production storage strategy

### 6.1 Primary recommendation: PostgreSQL

PostgreSQL is the source of truth because the core problem combines indexed filtering with transactional allocation:

- row-level locking provides a shared concurrency boundary across every API instance;
- `FOR UPDATE SKIP LOCKED` lets competing allocators skip rows already claimed by another transaction;
- one transaction can select, reserve, audit, and persist idempotency atomically;
- B-tree indexes and planner bitmap combinations support variable fixed digit positions;
- constraints enforce the state invariants close to the data;
- backups, replication, failover, monitoring, and operational tooling are mature;
- 10M records are within a normal relational operating range when schema, indexes, maintenance, and capacity are validated for the actual workload.

No unmeasured latency or throughput number is claimed. Dataset size alone is not enough to guarantee performance; representative benchmarks and query-plan review are required.

### 6.2 Alternative tradeoffs

| Option | Tradeoff |
| --- | --- |
| Redis only | Fast in-memory set/bitmap operations, but durability, authoritative reservation transactions, recovery, and audit consistency become more complex. It should not be the baseline correctness authority. |
| Elasticsearch/OpenSearch | Strong search capabilities, but distributed indexing is not a natural authority for single-winner transactional allocation; index refresh and consistency add risk. |
| MongoDB-style document store | Conditional updates can implement claims, but arbitrary positional predicate planning and multi-record audit/idempotency transactions are less direct than the selected relational design. |
| PostgreSQL plus Redis | Adds cache invalidation and operational complexity. It is justified only if measured database search cost remains unacceptable after indexing and query tuning. |

## 7. Pattern parsing and search algorithm

### 7.1 Validation and parsing

The parser performs exactly six iterations, so parser cost is `O(6)`, effectively constant:

```text
if character count != 6: reject

predicates = []
for position from 1 through 6:
    character = pattern[position]
    if character is '*': continue
    if character is not between '0' and '9': reject
    add equality predicate digit_position = numeric(character)
```

Validation rejects empty, shorter, longer, non-ASCII digit, whitespace, and unsupported-symbol patterns deterministically before any database query.

### 7.2 Pattern walkthroughs

`****23`:

1. Positions 1-4 are wildcards, so they add no predicates.
2. Position 5 adds `digit_5 = 2`.
3. Position 6 adds `digit_6 = 3`.

`1****5`:

1. Position 1 adds `digit_1 = 1`.
2. Positions 2-5 add no predicates.
3. Position 6 adds `digit_6 = 5`.

`123***`:

1. Positions 1-3 add `digit_1 = 1`, `digit_2 = 2`, and `digit_3 = 3`.
2. Positions 4-6 add no predicates.

`123456` is an exact match and can use `number = '123456'`. `******` has no digit predicate and means allocate from the eligible pool rather than return every ticket.

### 7.3 Candidate limiting

The allocation endpoint never materializes a full result set. It asks the database for one eligible candidate with `LIMIT 1`, locks it, and reserves it. If a future browse endpoint is required, it must use bounded keyset pagination and remain separate from allocation.

The application does not scan 10M rows or run wildcard matching over fetched records. PostgreSQL evaluates position predicates and availability filters using indexes and its query planner.

## 8. Indexing strategy

### 8.1 Baseline indexes

- unique primary B-tree index on `ticket_id`;
- non-unique B-tree index on `number` for exact matches and measured prefix cases;
- one B-tree index for each `digit_1` through `digit_6` position;
- availability-oriented indexes on state and reservation expiry, such as `(state, reserved_until, ticket_id)`;
- partial index for immediately available tickets where `state = 'AVAILABLE'`;
- unique indexes for active idempotency identities and reservation IDs where required.

PostgreSQL does not provide a persistent bitmap-index type. Its planner may combine multiple ordinary B-tree index scans with in-memory `BitmapAnd`/`BitmapOr` operations. For example, `digit_1 = 1 AND digit_6 = 5` can combine the position-1 and position-6 indexes.

### 8.2 Selectivity

Under a theoretical uniform digit distribution:

| Fixed positions | Estimated fraction | Estimated candidates among 10M before state filtering |
| --- | --- | --- |
| 6 | `1 / 1,000,000` | About 10 |
| 3 | `1 / 1,000` | About 10,000 |
| 2 | `1 / 100` | About 100,000 |
| 1 | `1 / 10` | About 1,000,000 |
| 0 (`******`) | `1` | Up to 10,000,000 |

These are probability estimates, not benchmark results. Real ticket distributions and available-state ratios may be skewed.

### 8.3 Composite indexes and hot patterns

Six positions have 63 non-empty subsets. Creating a composite index for every subset would impose excessive storage, cache, write, vacuum, and planning cost.

The baseline uses per-position indexes. Metrics and `EXPLAIN (ANALYZE, BUFFERS)` identify frequent slow shapes. Only then should a hot pattern receive a targeted composite or partial index, for example an index beginning with `digit_5, digit_6` if suffix searches dominate.

Every added index speeds some reads but increases insert/update cost, storage, memory pressure, and maintenance. Index count should be driven by observed workload rather than speculative completeness.

## 9. Atomic ticket reservation

The baseline reservation is one PostgreSQL transaction. SQL-like pseudocode illustrates the design but is not a shipped implementation:

```sql
BEGIN;

WITH candidate AS (
    SELECT ticket_id
    FROM tickets
    WHERE <fixed-digit predicates>
      AND (
          state = 'AVAILABLE'
          OR (state = 'RESERVED' AND reserved_until <= CURRENT_TIMESTAMP)
      )
    ORDER BY ticket_id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
UPDATE tickets AS ticket
SET state = 'RESERVED',
    reservation_id = :new_reservation_id,
    reserved_by = :requester_id,
    reserved_until = CURRENT_TIMESTAMP + :configured_ttl,
    consumed_reservation_id = NULL,
    consumed_by = NULL,
    consumed_at = NULL,
    updated_at = CURRENT_TIMESTAMP,
    version = version + 1
FROM candidate
WHERE ticket.ticket_id = candidate.ticket_id
RETURNING ticket.*;

INSERT INTO ticket_events (...);
INSERT INTO idempotency_records (...);

COMMIT;
```

The transaction commits before the API returns success. If no candidate is returned, the API reports no availability. If any step fails, rollback leaves the ticket unreserved and no partial event or idempotency result remains.

`SKIP LOCKED` lets competing transactions continue to another matching row rather than wait behind one locked hot row. The row lock is attached to `ticket_id`, so two overlapping patterns cannot reserve the same instance. Process-local mutexes are neither required nor sufficient.

### Candidate ordering

Deterministic `ticket_id` ordering is simple and testable but can concentrate contention near the start of an index range. `ORDER BY random()` is inappropriate for a large candidate set because it can force expensive sorting.

If measurements show hot-range contention, add a stable allocation bucket derived from `ticket_id`, choose a starting bucket per request, and scan buckets in bounded order. PostgreSQL remains the final atomic claimant regardless of ordering strategy.

## 10. Reservation expiry and recovery

- `reserved_until` is a lease evaluated against database time.
- Allocation treats expired `RESERVED` tickets as eligible and may reclaim them atomically.
- A periodic worker may normalize expired reservations to `AVAILABLE` and append expiry events in bounded batches.
- Correctness does not depend on that worker running on time.
- Cleanup must lock or conditionally update the current reservation so it cannot erase a newer reservation.
- `CONSUMED` is never made available by expiry cleanup.

Crash scenarios:

| Scenario | Expected outcome |
| --- | --- |
| API crashes before transaction commit | PostgreSQL rolls back; no reservation is visible. |
| API crashes after commit but before response | Reservation exists; client retry with the same idempotency key returns it. |
| Client disappears after receiving reservation | Lease expires and ticket becomes eligible again. |
| Consume and expiry/reclaim race | Row locking serializes contenders; only one valid transition commits. |
| Cleanup sees an old reservation | Reservation ID/version condition prevents clearing a newer reservation. |

## 11. Distributed concurrency and failure handling

| Scenario | Correctness behavior | Availability behavior |
| --- | --- | --- |
| Same pattern on multiple API instances | Shared PostgreSQL row locks and `SKIP LOCKED` give each committed request a different `ticket_id`. | Requests progress on different rows while candidates exist. |
| Overlapping patterns | The row lock is per ticket, so patterns cannot concurrently claim the same instance. | One request skips or retries another candidate. |
| Transaction statement fails | Entire transaction rolls back, including audit and idempotency writes. | Request fails and may be retried. |
| Request is canceled before commit | Transaction is canceled and rolled back. | Client receives cancellation/timeout. |
| Client times out with unknown commit result | Reuse the idempotency key to discover the original result. | A retry may wait for recovery but must not allocate another ticket. |
| PostgreSQL restart/failover | Uncommitted work rolls back; committed WAL-backed work remains authoritative after recovery according to database durability configuration. | Allocation is temporarily unavailable during failover. |
| API instance crashes | No correctness state is held only in its memory. | Load balancer routes new requests to healthy instances. |
| Reservation expires | It can be atomically reclaimed; stale owner cannot consume it. | Capacity returns without manual intervention. |
| Redis/cache is stale or unavailable | PostgreSQL revalidates eligibility and remains the allocation authority. | Fall back to PostgreSQL or fail the acceleration path; no duplicate allocation occurs. |

The design distinguishes correctness from availability: database failure may temporarily prevent allocation, but it must not cause duplicate allocation. Returning an error is safer than bypassing the transactional authority.

## 12. Performance and capacity analysis

The expected search size decreases as the pattern contains more fixed digits. The following figures assume digits are distributed evenly and are estimates, not benchmark results:

| Fixed digits | Example | Estimated matches in 10M records before availability filtering |
| ---: | --- | ---: |
| 6 | `123456` | About 10 |
| 3 | `123***` | About 10,000 |
| 2 | `****23` | About 100,000 |
| 1 | `1*****` | About 1,000,000 |
| 0 | `******` | Up to 10,000,000 |

The database does not return all matching rows. Allocation uses indexes, filters for eligible tickets, and stops after locking one row with `LIMIT 1`. Therefore, `******` means “reserve any available ticket,” not “return all 10 million tickets.”

Patterns with more fixed digits should be easier to narrow down. Broad patterns may inspect more index entries, and popular patterns may cause more lock competition. `SKIP LOCKED` helps concurrent requests move to another ticket instead of waiting for one locked row.

Before production, test at least:

1. patterns with 0, 1, 2, 3, and 6 fixed digits on 10M+ representative records;
2. many requests using the same pattern at the same time;
3. different patterns whose results overlap;
4. expired reservations and client retries;
5. the indexes and query plans with PostgreSQL `EXPLAIN ANALYZE`.

The API can be scaled to multiple instances because PostgreSQL protects the shared ticket rows. Database connections and transactions should be limited and kept short. Actual latency and throughput must come from testing rather than assumptions.

## 13. Optional future acceleration

Start with PostgreSQL only. This keeps the design easier to build, test, and operate.

If load tests later show that finding candidates is too slow, Redis may be added as a cache that suggests matching `ticket_id` values. PostgreSQL must still check the ticket and perform the final reservation because cached data may be old.

Redis is therefore an optional speed improvement, not the source of truth. It should not be added until measurements show a real need.
