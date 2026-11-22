# Agent documentation for tuipresent

These files are for LLM agents that write, check or answer questions about tuipresent decks. They describe the code as it is, quirks included.

## Read order

1. **Writing a deck from a brief:** start with [docs/agent-skill/SKILL.md](../agent-skill/SKILL.md) (condensed, self-contained workflow and rules), then [docs/AGENT_GUIDE.md](../AGENT_GUIDE.md) (full workflow, verification steps, layout budget, design guidance, pre-handoff checklist). Consult LANGUAGE.md for any construct you are unsure about: the guide links to it instead of restating syntax.
2. **Answering a precise question about the syntax or layout** (what does X parse to, how many rows does Y take, why did Z vanish): go straight to [LANGUAGE.md](LANGUAGE.md), then confirm with the matching case in [EXAMPLES.md](EXAMPLES.md).
3. **Validating front matter in an editor or tool:** [frontmatter.schema.json](frontmatter.schema.json).

## Which file answers what

| Question | File |
|---|---|
| How do I go from "make a talk about X" to a verified deck? | AGENT_GUIDE.md §1-2, SKILL.md |
| What is the exact grammar of a directive, note, image line or fence? | LANGUAGE.md §5, §8, §9, §10, §17 |
| In what order is each line checked? Why is my title not a title? | LANGUAGE.md §4, §7 |
| Which block appears at which reveal step? | LANGUAGE.md §6; EXAMPLES.md 7-10 |
| How wide is a column, how many rows do I have, how big will an image be? | LANGUAGE.md §11 (formulas and a table for 80x24, 100x30, 120x40), §9.5-9.6 |
| How is a `jump_to_middle` slide centered? | LANGUAGE.md §12; EXAMPLES.md 14-15 |
| What does `-check` catch, and what does it miss? | LANGUAGE.md §14 |
| What warns, what is an error, what fails silently? | LANGUAGE.md §19; EXAMPLES.md 17, 27 |
| Which front matter keys exist, with which types? | LANGUAGE.md §2; frontmatter.schema.json |
| Theme names, aliases, palette keys, theme files, painting | LANGUAGE.md §2.5, §18 |
| What does a construct actually look like on screen? | EXAMPLES.md (real `-dump` output at 80x24) |
| Design advice: how many bullets, which theme or transition, when to pause? | AGENT_GUIDE.md §4, §6 |

## Relationship between the docs

- **LANGUAGE.md** is the normative specification: rules, regexes, formulas, edge cases. It does not tell you how to design a talk.
- **EXAMPLES.md** is evidence: each case is a minimal deck plus its real `-check` line and `-dump` frames. `go test ./internal/docscheck` fails when any of that output differs from the renderer; after a parser or renderer change, run `go test ./internal/docscheck -run Examples -update` and review the diff. No case runs deck code.
- **frontmatter.schema.json** is the machine-readable front matter; `go test` checks its keys against the code.
- **AGENT_GUIDE.md** is the workflow guide: requirements, outline, verification commands, budgets and rules of thumb, design guidance, an annotated deck and a checklist. It links to LANGUAGE.md for syntax.
- **SKILL.md** is the condensed skill for agent runtimes that load skills. It is installed on its own, so it repeats the essential rules.
- **docs/guide/** is the human user guide; [reference.md](../guide/reference.md) is its complete reference.

If two documents disagree, the code wins. LANGUAGE.md and EXAMPLES.md are the most precise; fix the others (or report the drift).

## Ground rules for agents

- Always build and run `-check` and `-dump` at 80x24 and 100x30 before handing off a deck (AGENT_GUIDE.md §2).
- Never pass `-exec` without the user's approval: it runs the deck's code.
- Apart from load errors and a few warnings (LANGUAGE.md §19), authoring mistakes degrade silently. Fix every warning and read the dump.
