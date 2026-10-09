# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Battlebarge is a Go (Gin) REST API for tracking tabletop-wargame warbands, units (kills, XP, perks), and campaigns. Users authenticate via Firebase Auth; all application data lives in PostgreSQL. There are no tests, Makefile, or linter config in the repo yet.

## Repository layout

- `backend/` holds all the Go code (module `battlebarge`, so import paths are unchanged): `cmd/`, `controllers/`, `db/` (including `db/migrations/`), `middleware/`, `models/`, `repositories/`, `routes/`, `testutil/`, plus `go.mod`/`go.sum`. Paths in the architecture notes below are relative to `backend/` unless they say otherwise. Run Go commands from `backend/`.
- `docs/openapi.yaml` is the shared API contract and stays at the repo root so a frontend can generate types from it. `firebase.json` and `.firebaserc` (emulator and project config) also stay at the root.
- `frontend/` is planned and does not exist yet. It will have its own `.env`, separate from the backend's; never put backend secrets there.
- The backend's secrets live in `backend/.env` (each part of the project keeps its own `.env`, and nothing lives at the repo root). All `.env` files are protected: gitignored at any depth, `.claude/settings.json` denies reading or editing any `.env` or `.env.*` file at any depth, and a hook blocks tool calls that name one. Do not try to read them, including the frontend's. If a command needs their values, have the user run it, for example with the `!` prefix.

## Commands

- Run the server: `cd backend && go run ./cmd`. `main.go` loads `backend/.env` from the working directory or its parent if one exists (so it works from `backend/` or `backend/cmd/`, and never looks higher); real environment variables take precedence and a missing file is fine. Listens on `$PORT` (default `8080`).
- Build / vet (from `backend/`): `go build ./...`, `go vet ./...`
- Tests (from `backend/`): `go test ./...`; single test: `go test ./repositories -run TestAddAndDeleteUnitPerk`. Database tests create a throwaway schema in the Postgres named by `TEST_POSTGRES_URL` (e.g. `postgres://user@localhost:5432/bbtest`) and skip when it is unset. Tests sit beside the code they test; shared database fixtures are in `testutil/`. The table definitions live in `testutil/db.go` (`schemaSQL`) and must be kept in sync with the real schema. Controller tests use the real handlers with a fake auth middleware (`X-Test-UID` header); Firebase itself is never called.
- Firebase emulators (Auth on 9099, Firestore on 8000, UI enabled): `firebase emulators:start`
- `.env` (gitignored) needs `FIREBASE_PROJECT_ID`, `POSTGRES_URL`, optionally `REQUIRE_EMAIL_VERIFICATION` (default on; `false` disables it for local development), `TRUSTED_PROXIES` (comma-separated load balancer addresses/CIDRs allowed to set `X-Forwarded-For`; unset means client IPs come from the connection) and `CORS_ALLOWED_ORIGINS` (comma-separated browser origins allowed to call the API, e.g. `http://localhost:5173`; unset means no cross-origin access), and for local dev `FIREBASE_AUTH_EMULATOR_HOST` / `FIRESTORE_EMULATOR_HOST`. `testlogin.html` (gitignored) is a local helper for obtaining ID tokens.
- The Postgres schema is not in the repo; the SQL in `repositories/` is the source of truth for table/column names. `backend/db/migrations/` holds hand-run SQL files (001 unit_perks, 002 campaigns, 003 campaign join codes, 004 case-insensitive usernames; run each once, manually).

## Architecture

Layered, with one file per resource in each layer (`auth`, `user`, `warband`, `unit`):

`routes/` → `controllers/v1/` → `repositories/` → `db/` (global `db.PGClient` pgx pool, `db.AuthClient` Firebase auth client)

- `cmd/main.go` loads env, connects Firebase then Postgres (both panic on failure), then registers each `routes.Get<X>Controllers(r)` function. A new resource needs its routes function registered here.
- `routes/`: each resource gets a group; public routes are registered directly, authenticated ones go on a sub-group with `middleware.RequireAuth()`.
- `middleware/`: `RequireAuth()` verifies the Firebase Bearer token and sets `uid` in the Gin context (`middleware.ContextUIDKey`). `LoadUser()` (must be chained after `RequireAuth`) additionally loads the full `models.User` from Postgres into context (`ContextUserKey`). Use `RequireAuth` alone unless profile data is needed. `RequireAuth` returns 403 `email not verified` until the Firebase account's email is verified (set `REQUIRE_EMAIL_VERIFICATION=false` to disable, local development only). `RequireAuth` also rejects revoked/disabled accounts (one extra Firebase call per request); tests swap the verifier with `middleware.SetTokenVerifier`. `main.go` also installs `CORS`, `MaxBodySize` (64 KiB), and a per-IP `RateLimit`; `/auth/register` has its own stricter limit in `routes/authRoutes.go`.
- `controllers/v1/`: bind JSON into request structs from `models/`, read the uid from context, enforce ownership (e.g. `repositories.IsWarbandOwner` before creating a unit; returns 404 rather than 403 for non-owned resources), then call repositories.
- `repositories/`: raw SQL via pgx, no ORM. `Unit.Perks` lives in its own `unit_perks` table (FK to `units`, ON DELETE CASCADE); unit repository functions load perks after fetching units.
- `models/models.go`: all domain structs and request DTOs in one file. Campaigns: a campaign has chapters, any number of named/renamable teams, and member warbands (`campaign_warbands`, one team per warband per campaign; a warband can be in many campaigns). Anyone with a campaign ID can join a warband they own; only the campaign owner manages chapters and teams; the campaign owner or warband owner can move/remove a member. A team with warbands on it cannot be deleted (409). Joining needs the campaign's secret `join_code` (the campaign ID is public); only the owner can see it (create response, the owner's `GET /campaigns`, `GET /campaigns/:id/join-code`) and rotate it, and the repository queries never load it into a campaign, so public responses cannot carry it by accident. The owner can join without the code.
- IDs are `uuid.UUID` generated in controllers, except users, whose ID is the Firebase UID string.

## Validation and limits

Request structs in `models/models.go` carry the validation (binding tags): length caps, number ranges, `notblank` and `safetext` (custom tags in `models/validation.go`). Per-account record caps live in `repositories/limits.go` (`Max...` constants) and are enforced inside the insert's transaction while locking the parent row, so concurrent requests cannot exceed them; controllers turn `ErrLimitReached` into 409. When you add a field or a new kind of child record, add its validation tags and, if it can grow without bound, a cap. Keep `docs/openapi.yaml` in step with these limits.

## API documentation

`docs/openapi.yaml` (repo root, shared by backend and frontend) is the OpenAPI spec for the frontend. `backend/controllers/v1/openapi_test.go` fails if a route is added, removed, or renamed without updating the spec, or if a real response gains, loses, or nulls a field the schema lists. When you change an endpoint, request body, or response shape, update the spec in the same change.

## Documentation convention

Every function except `main` has a doc comment directly above it, so it shows on hover. It follows the Go convention that GoLand checks for: the comment **starts with the function's name**, as a short summary sentence, followed by the `Arguments` and `Returns` lines:

```go
// IncrementUnitXP adds to a unit's experience, clamping the result at 0
//
// Arguments: id (string) - unit ID; amount (int) - value to add, may be negative
//
// Returns: models.Unit - the updated unit; error - pgx.ErrNoRows if not found
func IncrementUnitXP(id string, amount int) (models.Unit, error) {
```

Gin handlers say what they handle in the summary and use `gin context` for arguments, listing the response status codes in place of returns:

```go
// AddUnitXP handles PATCH /units/:id/xp. Adds the requested amount to a unit's experience
//
// Arguments: gin context
//
// Returns: None (responds 200 with the updated unit; 400 on bad input; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
func AddUnitXP(c *gin.Context) {
```

Write the summary as a sentence that continues from the name ("IncrementUnitXP adds ...", "Handles ..."). Extra detail goes in further paragraphs under the summary, before `Arguments`. Test functions start with their name too ("TestX checks that ..."). Add this comment to every new function, and update it whenever a function's arguments, returns, or behavior change.

## TODO

- Production launch work, including the pending security decisions (account deletion, public owner IDs, enumeration, password policy, revocation cost, username lookalikes, bot protection), is tracked in `PRODUCTION_CHECKLIST.md`. Update it when you finish an item or discover a new launch requirement.
- Add Firebase Auth emulator tests (emulator config is in `firebase.json`, port 9099; skip when `FIREBASE_AUTH_EMULATOR_HOST` is unset, like the Postgres tests). Currently untested: a valid token being accepted by `RequireAuth`, the invalid/expired token branch, `LoadUser` with a verified token, and `RegisterUser` end to end (Firebase user creation, rollback when the Postgres insert fails, 409 on duplicate email). Controller tests use a fake `X-Test-UID` auth middleware instead of real token verification.
