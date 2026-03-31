# Manage Backend (Phase 1 Foundation)

This repository currently implements the phase-1 foundation:

- user/class core models
- 4-level RBAC action authorization
- scope-aware repositories
- header-based identity middleware
- student `/api/v1/me` endpoint
- admin users/classes/logs endpoints

## Run Tests

```bash
go test ./... -count=1
```

## API Reference

See [docs/api/phase1-foundation-api.md](docs/api/phase1-foundation-api.md).

