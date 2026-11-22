---
title: Acme Platform Kickoff
sub_title: A deck in a custom theme
author: Platform Team
theme:
  path: themes/brand.yaml
  background: paint
transition: wipe
---

Where we are
===

* Three services, one on-call rotation
* p99 latency down **38%** since spring
* Deploys: weekly → daily

> Ship small, ship often, sleep at night.

<!-- speaker_note: The theme comes from examples/themes/brand.yaml. -->

<!-- end_slide -->

Next quarter
===

| Goal | Owner | Target |
|------|-------|--------|
| Self-serve deploys | Platform | Sep |
| Error budgets per team | SRE | Oct |
| Retire the old queue | Data | Nov |

<!-- end_slide -->

How this deck is themed
===

<!-- column_layout: [1, 1] -->
<!-- column: 0 -->

```yaml
# themes/brand.yaml
extends: nord
accent: "#ff6b35"
accent2: "#f7c548"
code_style: monokai
```

<!-- column: 1 -->

```yaml
# front matter
theme:
  path: themes/brand.yaml
  background: paint
```

<!-- reset_layout -->

Edit the theme file while presenting: it reloads like the deck does.
