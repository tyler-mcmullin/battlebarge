# Production launch checklist

Tracks what is needed before launching Battlebarge. Check items off (`[x]`) as they are done and add new ones as they come up. Items marked **code** need a change in this repo; the rest are configuration or operations.

## Blockers

- [ ] **CORS origins (config).** The middleware is done; set `CORS_ALLOWED_ORIGINS` in the production environment to the real frontend origin(s), e.g. `https://app.example.com`. Use `https` and exact origins, no `localhost`, no trailing slash. If unset, browsers are blocked from calling the API.
- [ ] **Remove the Firebase emulator settings from production.** `FIREBASE_AUTH_EMULATOR_HOST` and `FIRESTORE_EMULATOR_HOST` must not be set, or the server talks to a local emulator instead of Firebase.
- [ ] **Firebase admin credentials.** `RegisterUser` creates and deletes Firebase users, which needs service-account credentials (`GOOGLE_APPLICATION_CREDENTIALS` or the platform's default credentials) for the production project. Token verification alone does not. Set `FIREBASE_PROJECT_ID` to the production project.
- [ ] **Production Postgres.** Managed instance, `POSTGRES_URL` with TLS (`sslmode=require` or stricter), and a dedicated least-privilege database user.
- [ ] **Run migrations on production, in order, once each.** 001 (unit_perks) and 002 (campaigns) in `db/migrations/`. There is no migration runner, so record here when each ran. Take a `pg_dump` backup first.
- [ ] **Release mode (config).** Set `GIN_MODE=release`.
- [ ] **Hosting and deployment.** Nothing exists yet (no Dockerfile or deploy config). Choose a host, build and deploy, and put it behind HTTPS (TLS terminated by the platform or a proxy).

## Security

- [ ] **Trusted proxies (code).** Behind a load balancer, call `SetTrustedProxies` so client IPs (logs, rate limits) are real, not the proxy's.
- [ ] **Rate limiting (code).** `POST /auth/register` and the other endpoints are open to abuse. Add per-IP limits, at least on registration. Check Firebase's own limits and consider email verification.
- [ ] **Server timeouts and graceful shutdown (code).** `r.Run` sets no read, write or idle timeouts and does not drain requests on shutdown. Use an `http.Server` with timeouts and `Shutdown`.
- [ ] **Request body size limit (code).** Cap JSON body size.
- [ ] **Firebase console.** Add the production domain to authorized domains; review password policy and sign-in methods.
- [ ] **Review what is public.** `GET /warbands/:id`, `/units/:id` and `/campaigns/:id` need no login and return owner IDs. IDs are random UUIDs, but confirm public-by-ID is intended.
- [x] `.env` is gitignored and was never committed (checked git history).
- [ ] **Secrets management.** Store production secrets in the platform's secret store, not in files. Rotate anything that was ever shared.

## Operations

- [ ] **Health check endpoint (code).** Add `GET /healthz` (including a database ping) for the load balancer.
- [ ] **Backups.** Automated Postgres backups, plus a tested restore.
- [ ] **Logging and monitoring.** Unexpected errors are logged with `log.Printf` and returned as a generic 500. Ship logs somewhere searchable and alert on 5xx rates.
- [ ] **Database connection pool.** Tune pgx pool size to the database's connection limit.
- [ ] **CI (code).** GitHub Action running `go vet` and `go test ./...` against a Postgres service so the database tests run on every push.

## Quality gaps

- [ ] **Firebase emulator tests (code).** Valid token accepted, invalid/expired token rejected, `RegisterUser` end to end including rollback and the duplicate-email 409. See `CLAUDE.md` TODO.
- [ ] **Account deletion.** No delete-user endpoint, and `warbands.user_id` has no `ON DELETE` action, so deleting a user with warbands fails. Needed for account deletion requests.
- [ ] **Profile management.** No endpoint to change username or email.
- [ ] **Pagination.** List endpoints return everything; fine at launch size.
- [ ] **API documentation.** OpenAPI spec or endpoint reference for the frontend.
- [ ] **Schema in the repo.** Keep a current schema file so `testutil/db.go` and the real database cannot drift.

## Done

- [x] `.env` is optional: `main.go` loads it from the working directory or its parent when present, real environment variables win, and a missing file is fine (a malformed one still stops startup).
- [x] Port comes from `PORT` (default 8080).
- [x] CORS middleware (`middleware.CORS`, configured by `CORS_ALLOWED_ORIGINS`).
- [x] Malformed IDs return 404/400, and raw database errors are not sent to clients.
- [x] Test suite for repositories, controllers, middleware, and routes.
- [x] Migration 001 (unit_perks) applied to the development database.
