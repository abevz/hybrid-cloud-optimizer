# Specs

This directory contains specification-first development documents.

Each spec lives in a numbered directory:

```text
docs/specs/
  001-mvp/
    README.md
    constitution.md
    glossary.md
    spec.md
    requirements.md
    design.md
    detailed-design.md
    tasks.md
    testing.md
    traceability.md
```

Use this flow for changes:

1. Read the spec directory `README.md` for document order and working rules.
2. Update the relevant spec document first.
3. If a contract changes, update `detailed-design.md`.
4. Refresh `traceability.md` in the same change.
5. Implement the smallest matching task.
6. Add or update tests for that task.
7. Open a Pull Request that references the spec task.

The root `MVP.md` can remain as the legacy source document while specs are
gradually split into smaller files.
