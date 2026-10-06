# Production launch checklist

Tracks what is needed before launching Battlebarge. Check items off (`[x]`) as they are done and add new ones as they come up. Items marked **code** need a change in this repo; the rest are configuration or operations.

## Blockers

- [ ] **CORS origins (config).** The middleware is done; set `CORS_ALLOWED_ORIGINS` in the production environment to the real frontend origin(s), e.g. `https://app.example.com`. Use `https` and exact origins, no `localhost`, no trailing slash. If unset, browsers are blocked from calling the API.
- [ ] **Remove the Firebase emulator settings from production.** `FIREBASE_AUTH_EMULATOR_HOST` and `FIRESTORE_EMULATOR_HOST` must not be set, or the server talks to a local emulator instead of Firebase.
- [ ] **Firebase admin credentials.** `RegisterUser` creates and deletes Firebase users, which needs service-account credentials (`GOOGLE_APPLICATION_CREDENTIALS` or the platform's default credentials) for the production project. Token verification alone does not. Set `FIREBASE_PROJECT_ID` to the production project.
- [ ] **Production Postgres.** Managed instance, `POSTGRES_URL` with TLS (`sslmode=require` or stricter), and a dedicated least-privilege database user.
- [ ] **Run migrations on production, in order, once each.** 001 (unit_perks), 002 (campaigns), 003 (campaign join codes) and 004 (case-insensitive usernames) in `db/migrations/`. There is no migration runner, so record here when each ran: development has had 001 only so far. Take a `pg_dump` backup first. 003 gives every existing campaign a random join code (owners can rotate it). 004 refuses to run, with a message naming them, if two usernames differ only by case; rename those users first.
- [ ] **Email verification (config).** The API requires a verified email (403 `email not verified` otherwise). Verification lives in Firebase Authentication, not in Postgres: the frontend calls `sendEmailVerification`, Firebase emails a one-time link, and clicking it makes the handler call `applyActionCode`, which sets `emailVerified` on the Firebase account. The API only reads the signed `email_verified` claim in the ID token, so a token issued before verifying stays unverified until the frontend refreshes it (`user.reload()` then `getIdToken(true)`), and changing the account's email resets it. Before launch:
  - [ ] **Sender and spam.** The default Firebase sender often lands in spam. Configure a custom sender domain or SMTP and customize the template (Firebase console, Authentication, Templates).
  - [ ] **Authorized domains.** Add the frontend's domain in the console, or the links will not work.
  - [ ] **Action URL.** Decide whether the link opens Firebase's hosted page or a page on your own site (custom action URL), which allows a proper "email verified, continue" screen.
  - [ ] **Frontend flow.** After sign-up, show "check your inbox", offer a resend button, and refresh the ID token after the user verifies. Handle the 403 `email not verified` response by prompting to verify rather than signing out.
  - [ ] **Not disabled.** `REQUIRE_EMAIL_VERIFICATION` must not be set to `false` in production (local development only; the server logs a warning when it is).
  - Accounts created through social providers such as Google arrive already verified; only email-and-password accounts go through this flow.
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

## Security decisions pending

Decided later; each needs a product call. Options and my recommendation are summarized here.

- [ ] **Account deletion.** No way to delete an account, and deleting a user who has warbands is blocked by the database (`warbands.user_id` has no `ON DELETE` action). Privacy laws such as GDPR can require deletion on request. Decide what happens to a deleted user's campaigns: delete them (disrupts other players), or require transferring or deleting them first (needs a transfer-ownership endpoint). Recommended: delete the user's warbands and units, and require handing off or deleting owned campaigns first.
- [ ] **Public data and owner IDs.** Warbands, units and campaigns are readable by anyone with the ID, and responses include the owner's Firebase UID (`user_id`, `owner_id`). Options: leave as is; swap the UID for the owner's username in public responses (probably what the frontend wants to display anyway); or add per-item visibility (`public` / `members only`), which is a bigger permissions model. Recommended: keep the unlisted-link model and swap UIDs for usernames when the frontend needs owner names.
- [ ] **Account enumeration.** Registration says "email already exists" / "username already taken", so anyone can test whether an email has an account (the register rate limit slows bulk probing). Options: keep specific messages (best experience); generic error for emails only; or fully hide it ("check your inbox" either way), which needs email sending. Recommended: keep for now and revisit when you add email sending.
- [ ] **Password policy.** Firebase's default minimum is 6 characters; the API can raise it at registration, but Firebase's client-side change/reset would bypass that. Recommended: require at least 10 characters in both places (API check plus the Firebase console password policy), no complexity rules. Optionally check against breached-password lists.
- [ ] **Revocation check cost.** `RequireAuth` calls Firebase on every request to detect revoked or disabled accounts. Options: check every request (now); a 30-60 second cache; check only on writes; or no check (up to an hour of access after revocation). Recommended: keep as is until latency is measured, then add the short cache.
- [ ] **Username characters and lookalikes.** Usernames are now unique ignoring case, but lookalike characters (for example a Cyrillic "а" for "a") and confusing punctuation are still allowed. Options: restrict to letters, digits, `_`, `-`, `.`; or normalize Unicode and reject confusables. Restricting is simpler but excludes non-Latin names.
- [ ] **Bot protection on registration.** Verified email limits the damage from junk accounts but does not stop them being created. If bots show up, add Firebase App Check or a CAPTCHA.

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
- [x] Required email verification: authenticated routes return 403 until the account's email is verified.
- [x] Usernames are unique ignoring case (migration 004), and registration conflicts are reported from the constraint that failed, so an email conflict is no longer called a username one. A failed Firebase rollback during registration is now logged.
- [x] Security scan fixes: Go 1.26.6 and updated grpc, x/net, x/text, quic-go, and otel/sdk (`govulncheck` reports no reachable vulnerabilities); request body cap (64 KiB, 413); field length and number range validation with blank/control-character rejection; per-account record quotas enforced under a row lock (409); per-IP rate limit (20/s, burst 60) and a tighter one on registration (5, then 1/min, 429); server read/write/idle timeouts and graceful shutdown; trusted proxies off by default; revoked and disabled tokens rejected.
- [x] OpenAPI spec in `docs/openapi.yaml`, with tests that keep it in step with the routes and response shapes.
- [x] `.env` is optional: `main.go` loads it from the working directory or its parent when present, real environment variables win, and a missing file is fine (a malformed one still stops startup).
- [x] Port comes from `PORT` (default 8080).
- [x] CORS middleware (`middleware.CORS`, configured by `CORS_ALLOWED_ORIGINS`).
- [x] Malformed IDs return 404/400, and raw database errors are not sent to clients.
- [x] Test suite for repositories, controllers, middleware, and routes.
- [x] Migration 001 (unit_perks) applied to the development database.
