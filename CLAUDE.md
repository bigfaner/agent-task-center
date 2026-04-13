---
name: agent-task-center
description: Centralized task visualization & collaboration service for AI agents
paths:
  - server/cmd/server/main.go
  - server/internal/
  - web/src/
  - docs/features/
  - Makefile
---

# Agent Task Center

Monorepo: Go server (REST API) + React web UI (read-only kanban + file upload).

## Commands

```bash
make build          # build both server + web
make dev            # run dev-server + dev-web in parallel
make dev-server     # cd server && go run ./cmd/server
make dev-web        # cd web && npm run dev
make vet            # cd server && golangci-lint run ./...
make fmt            # cd server && golangci-lint fmt ./...
make lint           # cd web && npm run lint
make format         # cd web && npm run format
make check          # run vet + lint
cd server && go test ./...
cd web && npm run build
```

## Architecture

```
handler/ → service/ → db/        (Go, strict layer order)
parser/  (index.json / proposal.md / manifest.md)
web/src/pages/ → components/ → api/   (React)
```

**Server** (Go, chi router): SQLite (local) / PostgreSQL (prod). `DB_DRIVER` env var switches.
**Web** (React 19 + Vite): Tailwind CSS v4, shadcn/ui, react-query, react-router-dom. Builds to `server/web/dist` for embed.FS.

## Key Paths

| Path | Purpose |
|------|---------|
| `server/cmd/server/main.go` | Server entry point |
| `server/internal/{config,db,handler,model,parser,service}/` | Go layers |
| `web/src/` | React app |
| `docs/features/<slug>/manifest.md` | Feature index & traceability |
| `docs/features/<slug>/tasks/index.json` | Task definitions |

## Conventions

- **Go**: `internal/` package layout. No code outside `internal/` except `cmd/`. Use `sqlx` for DB, `chi` for routing.
- **React**: Pages in `src/pages/`, shared components in `src/components/`, API calls in `src/api/`. Use `@` path alias.
- **Docs**: Feature docs live in `docs/features/<slug>/` with manifest as entry point. Task records go through `task record` CLI.
- **Migrations**: `golang-migrate` in `server/internal/db/`.
- **API routes**: `/api/*` for web UI, `/api/agent/*` for CLI/agent operations.

## Working Rules

- Run `make vet` after Go changes (uses golangci-lint, config in server/.golangci.yml)
- Run `npm run build` after web changes to verify compilation
- Follow existing layer boundaries: handler never accesses db directly
