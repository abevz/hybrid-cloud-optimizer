# Specs

This directory contains specification-first development documents.

Each spec lives in a numbered directory:

```text
docs/specs/
  001-mvp/
    README.md
    spec.md
    requirements.md
    design.md
    tasks.md
    testing.md
```

Use this flow for changes:

1. Update the relevant spec.
2. Implement the smallest matching task.
3. Add or update tests for that task.
4. Open a Pull Request that references the spec task.

The root `MVP.md` can remain as the legacy source document while specs are
gradually split into smaller files.
