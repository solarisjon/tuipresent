---
title: "INC-2291 Postmortem"
sub_title: Checkout p99 hit 4.2 s for 47 minutes
author: Payments Platform
date: 2026-11-18
theme: dracula
transition: wipe
---

What happened
===

On Nov 18 a routine deploy cold-started the price cache during peak
traffic. Retries turned the slowdown into a database overload.

| Impact | Value |
|--------|-------|
| Duration | 47 min (14:02 to 14:49 UTC) |
| Peak checkout p99 | 4.2 s (SLO 300 ms) |
| Checkouts timed out | 3.1% (about 1,900 orders) |
| Data loss | None |

<!-- pause -->

This review is blameless. We are fixing the system, not the people.

<!-- end_slide -->

Timeline (UTC)
===

| Time | Event |
|------|-------|
| 14:02 | Deploy 4471 ships, includes cache key change |
| 14:06 | Price-cache hit rate falls from 94% to 31% |
| 14:09 | p99 alert fires, on-call paged |
| 14:13 | DB connection pool exhausted (400 / 400) |
| 14:21 | Deploy 4471 suspected, rollback started |
| 14:35 | Rollback complete, cache refilling |
| 14:49 | p99 under 250 ms, incident resolved |

<!-- pause -->

Detection took 7 minutes. Suspecting the deploy took 12 more.

<!-- end_slide -->

Checkout p99 latency
===

```text
5-min bucket   p99 (ms), one cell = 105 ms        SLO 300 ms
13:55  ██                                          180
14:00  ██                                          190
14:05  ████████                                    820
14:10  ██████████████████████                    2,310
14:15  ████████████████████████████████████████  4,200
14:20  █████████████████████████████████████     3,900
14:25  ██████████████████████████████████        3,600
14:30  ████████████████████████████              2,900
14:35  ██████████████                            1,500
14:40  ██████                                      610
14:45  ███                                         290
14:50  ██                                          200
```

<!-- end_slide -->

The same service, 20 minutes apart
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Before, 13:55

```text
cache hit rate     94%
db cpu             41%
db connections     120 / 400
client retries/s   12
checkout p99       180 ms
```

<!-- column: 1 -->

## During, 14:15

```text
cache hit rate     31%
db cpu             98%
db connections     400 / 400
client retries/s   2,940
checkout p99       4,200 ms
```

<!-- reset_layout -->

<!-- pause -->

Once the pool was full, every request queued behind every other one.

<!-- end_slide -->

Why it happened
===

**Symptom.** Checkout p99 reached 4.2 s and 3.1% of checkouts timed out.

<!-- pause -->

**Trigger.** Deploy 4471 added `locale` to the price-cache key. Every
key was new, so the cache was effectively empty at peak traffic.

<!-- pause -->

**Amplifier.** Clients retried 3 times with no backoff and no budget.
Each slow request became up to four database reads.

<!-- pause -->

**Root cause.** Nothing in our process treats a cache key change as a
capacity change. No hit-rate alert, and the canary only watched errors.

<!-- end_slide -->

Retries multiplied the load 27x
===

```text
db reads/s = requests/s x miss rate x (1 + avg retries)

before   6,000 x 0.06 x 1.0  =     360
during   6,000 x 0.69 x 2.4  =   9,936
                                 ------
db read capacity             ~   2,500
```

<!-- pause -->

* Misses alone: 4,140 reads/s, 1.7x capacity. Slow.
* Misses plus retries: 9,936 reads/s, 4x capacity. Down.

<!-- end_slide -->

The change that shipped
===

```diff
 // pricing/cache.go
 func priceKey(sku string, r *Request) string {
-    return fmt.Sprintf("price:%s:%s", r.Region, sku)
+    return fmt.Sprintf("price:%s:%s:%s", r.Region, r.Locale, sku)
 }
```

<!-- pause -->

A correct change. Prices do vary by locale in two new markets. The
review approved the code; nobody asked what it would do to the cache.

<!-- end_slide -->

The fix, part one: bounded retries
===

```diff
 # clients/http/defaults.yaml
 retry:
-  attempts: 3
-  backoff: none
+  attempts: 2
+  backoff: exponential(base=50ms, factor=2, jitter=full)
+  budget: 10%            # of requests in a 10 s window
```

<!-- pause -->

The same miss rate now tops out near 4,700 reads/s, not 9,936.
Still over capacity, which is why part two is prewarming the cache.

<!-- end_slide -->

What went well, what didn't
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Went well

* Alert fired within 3 minutes
* Rollback was one command
* Status page up by 14:15
* No data loss or double charges

<!-- column: 1 -->

## Didn't

* Deploy suspected after 12 min
* Rollback took 14 minutes
* Cache refill never load-tested
* Retry storm hid the real error

<!-- end_slide -->

Action items
===

| Action | Owner | Due |
|--------|-------|-----|
| Alert when cache hit rate < 80% | Observability | Nov 25 |
| Runbook: cache cold start | On-call | Nov 28 |
| Retry budget + backoff in client lib | Platform | Dec 2 |
| Canary gates on hit rate and p99 | Release Eng | Dec 9 |
| Prewarm cache before key changes | Payments | Dec 9 |
| Show deploys on the latency dashboard | Observability | Dec 9 |

<!-- end_slide -->

<!-- jump_to_middle -->
<!-- transition: dissolve -->

A cache key change is a capacity change.
===

<!-- pause -->

Questions?
