# Production launch checklist

Tracks what is needed before launching Battlebarge. Check items off (`[x]`) as they are done and add new ones as they come up. Items marked **code** need a change in this repo; the rest are configuration or operations.

## Blockers

- [ ] **CORS origins (config).** The middleware is done; set `CORS_ALLOWED_ORIGINS` in the production environment to the real frontend origin(s), e.g. `https://app.example.com`. Use `https` and exact origins, no `localhost`, no trailing slash. If unset, browsers are blocked from calling the API.
- [ ] **Remove the Firebase emulator settings from production.** `FIREBASE_AUTH_EMULATOR_HOST` and `FIRESTORE_EMULATOR_HOST` must not be set, or the server talks to a local emulator instead of Firebase.
- [ ] **Firebase admin credentials.** `RegisterUser` creates and deletes Firebase users, which needs service-account credentials (`GOOGLE_APPLICATION_CREDENTIALS` or the platform's default credentials) for the production project. Token verification alone does not. Set `FIREBASE_PROJECT_ID` to the production project.
- [ ] **Production Postgres.** Managed instance, `POSTGRES_URL` with TLS (`sslmode=require` or stricter), and a dedicated least-privilege database user.
- [ ] **Run migrations on production, in order, once each.** 001 (unit_perks), 002 (campaigns) and 003 (campaign join codes) in `db/migrations/`. There is no migration runner, so record here when each ran: development has had 001 only so far. Take a `pg_dump` backup first. 003 gives every existing campaign a random join code (owners can rotate it).
- [ ] **Release mode (config).** Set `GIN_MODE=release`.
- [ ] **Hosting and deployment.** Nothing exists yet (no Dockerfile or deploy config). Choose a host, build and deploy, and put it behind HTTPS (TLS terminated by the platform or a proxy).

## Security

- [ ] **Set `TRUSTED_PROXIES` (config).** Rate limiting keys on the client IP. With the variable unset, the server ignores `X-Forwarded-For` and uses the connection's address, which is safe but means every client behind a load balancer shares one bucket. Set it to the load balancer's address(es) or CIDR range(s), comma-separated.
- [ ] **Rate limiting across instances.** Limits are held in memory per server instance. If you run several instances, each has its own counters; move to a gateway or shared store if that matters.
- [ ] **Firebase revocation check cost.** `RequireAuth` calls Firebase on every request to check for revoked or disabled accounts. Watch latency and Firebase quota; if it hurts, cache results for a short time.
- [ ] **Re-run `govulncheck` before each release** (`go run golang.org/x/vuln/cmd/govulncheck@latest ./...`) and build with Go 1.26.6 or newer.
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
- [ ] **Schema in the repo.** Keep a current schema file so `testutil/db.go` and the real database cannot drift.

## Done

- [x] Campaign join codes: joining needs the campaign's secret code (not just its public ID); only the owner sees it, and can rotate it. Join attempts have their own tight per-IP limit.
- [x] Security scan fixes: Go 1.26.6 and updated grpc, x/net, x/text, quic-go, and otel/sdk (`govulncheck` reports no reachable vulnerabilities); request body cap (64 KiB, 413); field length and number range validation with blank/control-character rejection; per-account record quotas enforced under a row lock (409); per-IP rate limit (20/s, burst 60) and a tighter one on registration (5, then 1/min, 429); server read/write/idle timeouts and graceful shutdown; trusted proxies off by default; revoked and disabled tokens rejected.
- [x] OpenAPI spec in `docs/openapi.yaml`, with tests that keep it in step with the routes and response shapes.
- [x] `.env` is optional: `main.go` loads it from the working directory or its parent when present, real environment variables win, and a missing file is fine (a malformed one still stops startup).
- [x] Port comes from `PORT` (default 8080).
- [x] CORS middleware (`middleware.CORS`, configured by `CORS_ALLOWED_ORIGINS`).
- [x] Malformed IDs return 404/400, and raw database errors are not sent to clients.
- [x] Test suite for repositories, controllers, middleware, and routes.
- [x] Migration 001 (unit_perks) applied to the development database.
