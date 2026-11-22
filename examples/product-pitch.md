---
title: Hushpage
sub_title: Alerts that know who to wake (a fictional product pitch)
author: Hushpage founding team
date: Seed round, 2026
theme: tokyonight-storm
transition: dissolve
---

<!-- jump_to_middle -->

3:12 a.m.
===

<!-- pause -->

Your phone buzzes.

<!-- speaker_note: Let the silence sit for a beat before pressing space. -->

<!-- end_slide -->

<!-- jump_to_middle -->

The fourth page this week.
===

<!-- pause -->

None of them were yours to fix.

<!-- end_slide -->

<!-- jump_to_middle -->
<!-- transition: cover -->

71%
===

of pages go to someone who cannot fix the problem.

<!-- pause -->

*Design-partner survey, 212 on-call engineers, 2026.*

<!-- speaker_note:
- Say the number, then stop talking
- If asked: the survey ran across all six design partners in March
-->

<!-- end_slide -->

Rotas page people. Problems belong to code.
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

## Today

* Alert fires on a service
* Rota wakes whoever is up
* They find the owner
* The owner fixes it

<!-- column: 1 -->

## With Hushpage

* Alert fires on a service
* We find the change behind it
* We wake its author's team
* They fix it

<!-- end_slide -->

<!-- jump_to_middle -->

Route by ownership, not by schedule.
===

<!-- end_slide -->

How it works
===

<!-- transition: cover -->

1. **Listen.** Every alert, deploy and feature flag flip.

<!-- pause -->

2. **Correlate.** Match the alert to the change most likely behind it.

<!-- pause -->

3. **Route.** Page the team that owns that change. Only them.

<!-- pause -->

No new agent. Connects to the alerting and CI you already run.

<!-- speaker_note:
Correlation is the hard part. Mention the **deploy + flag** signals
by name; skip the model details unless someone asks.
-->

<!-- end_slide -->

Eight weeks with six design partners
===

```text
pages per engineer per week

before   ██████████████████████████████████  14.2
after    ███████▏                             3.0

minutes to mitigate (median)

before   ██████████████████████████████████  42
after    ██████████████▌                      18
```

<!-- speaker_note: Pages dropped 79%. Mitigation time fell by more than half. -->

<!-- end_slide -->

<!-- jump_to_middle -->

Fewer pages. Faster fixes.
===

<!-- pause -->

Because the person who gets woken up already knows the code.

<!-- end_slide -->

The ask
===

<!-- transition: cover -->

* **$4M seed** to reach 50 paying teams
* **18 months** of runway
* **3 hires:** two integration engineers, one founding seller

<!-- pause -->

Pilot spots open now: 8 weeks, free, one team.

<!-- speaker_note:
- Two of the six partners have already signed letters of intent
- Close by asking who on their team gets paged the most
-->

<!-- end_slide -->

<!-- jump_to_middle -->

Let everyone else sleep.
===

<!-- pause -->

hushpage.example
