# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Battlebarge is a Go (Gin) REST API for tracking tabletop-wargame warbands, units (kills, XP, perks), and campaigns. Users authenticate via Firebase Auth; all application data lives in PostgreSQL. There are no tests, Makefile, or linter config in the repo yet.

## Commands

- Run the server: `cd cmd && go run .` — it must be run from `cmd/` because `main.go` loads `../.env`. Listens on `:8080`.
- Build / vet: `go build ./...`, `go vet ./...`
- Tests: `go test ./...`; single test: `go test ./repositories -run TestAddAndDeleteUnitPerk`. Database tests create a throwaway schema in the Postgres named by `TEST_POSTGRES_URL` (e.g. `postgres://user@localhost:5432/bbtest`) and skip when it is unset. Tests sit beside the code they test; shared database fixtures are in `testutil/`. The table definitions live in `testutil/db.go` (`schemaSQL`) and must be kept in sync with the real schema. Controller tests use the real handlers with a fake auth middleware (`X-Test-UID` header); Firebase itself is never called.
- Firebase emulators (Auth on 9099, Firestore on 8000, UI enabled): `firebase emulators:start`
- `.env` (gitignored) needs `FIREBASE_PROJECT_ID`, `POSTGRES_URL`, optionally `CORS_ALLOWED_ORIGINS` (comma-separated browser origins allowed to call the API, e.g. `http://localhost:5173`; unset means no cross-origin access), and for local dev `FIREBASE_AUTH_EMULATOR_HOST` / `FIRESTORE_EMULATOR_HOST`. `testlogin.html` (gitignored) is a local helper for obtaining ID tokens.
- The Postgres schema is not in the repo; the SQL in `repositories/` is the source of truth for table/column names. `db/migrations/` holds hand-run SQL files (001 unit_perks, 002 campaigns; run each once, manually).

## Architecture

Layered, with one file per resource in each layer (`auth`, `user`, `warband`, `unit`):

`routes/` → `controllers/v1/` → `repositories/` → `db/` (global `db.PGClient` pgx pool, `db.AuthClient` Firebase auth client)

- `cmd/main.go` loads env, connects Firebase then Postgres (both panic on failure), then registers each `routes.Get<X>Controllers(r)` function. A new resource needs its routes function registered here.
- `routes/`: each resource gets a group; public routes are registered directly, authenticated ones go on a sub-group with `middleware.RequireAuth()`.
- `middleware/`: `RequireAuth()` verifies the Firebase Bearer token and sets `uid` in the Gin context (`middleware.ContextUIDKey`). `LoadUser()` (must be chained after `RequireAuth`) additionally loads the full `models.User` from Postgres into context (`ContextUserKey`). Use `RequireAuth` alone unless profile data is needed.
- `controllers/v1/`: bind JSON into request structs from `models/`, read the uid from context, enforce ownership (e.g. `repositories.IsWarbandOwner` before creating a unit; returns 404 rather than 403 for non-owned resources), then call repositories.
- `repositories/`: raw SQL via pgx, no ORM. `Unit.Perks` lives in its own `unit_perks` table (FK to `units`, ON DELETE CASCADE); unit repository functions load perks after fetching units.
- `models/models.go`: all domain structs and request DTOs in one file. Campaigns: a campaign has chapters, any number of named/renamable teams, and member warbands (`campaign_warbands`, one team per warband per campaign; a warband can be in many campaigns). Anyone with a campaign ID can join a warband they own; only the campaign owner manages chapters and teams; the campaign owner or warband owner can move/remove a member. A team with warbands on it cannot be deleted (409).
- IDs are `uuid.UUID` generated in controllers, except users, whose ID is the Firebase UID string.

## Documentation convention

Every function except `main` has a doc comment directly above it in this format, so it shows on hover:

```go
// Arguments: id (string) - unit ID; amount (int) - value to add
//
// Returns: models.Unit - the updated unit; error - pgx.ErrNoRows if not found
//
// Short description of what the function does
```

Gin handlers use `gin context` for arguments and list the HTTP method/path and response status codes in place of returns. Add this comment to every new function, and update it whenever a function's arguments, returns, or behavior change.

## TODO

- Add Firebase Auth emulator tests (emulator config is in `firebase.json`, port 9099; skip when `FIREBASE_AUTH_EMULATOR_HOST` is unset, like the Postgres tests). Currently untested: a valid token being accepted by `RequireAuth`, the invalid/expired token branch, `LoadUser` with a verified token, and `RegisterUser` end to end (Firebase user creation, rollback when the Postgres insert fails, 409 on duplicate email). Controller tests use a fake `X-Test-UID` auth middleware instead of real token verification.
