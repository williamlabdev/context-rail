# Project Architecture

## Scope and boundary

This Project owns a small Python report generator and its synthetic fixtures. The upstream ticket system remains the source of truth and is read-only from this Project.

```text
Synthetic ticket fixture
        ↓
Python report generator
        ↓
Read-only report artifact
```

## Security and failure behavior

- No credentials or personal data are included.
- An absent or malformed fixture fails the report instead of generating partial claims.
- Production target is protected and not used by this fixture.

## Change impact checklist

Changes must assess source schema, metric definitions, privacy classification, test fixtures, report reproducibility and staging configuration.
