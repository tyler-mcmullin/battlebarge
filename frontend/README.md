# Battlebarge frontend

Angular 22 single-page app for Battlebarge. It talks to the Go API in `../backend` and signs users in with Firebase Authentication.

## Run it locally

You need three things running:

1. The Firebase Auth emulator: `firebase emulators:start` (from the repo root).
2. The Go API on port 8080, with `FIREBASE_AUTH_EMULATOR_HOST` pointing at the emulator (see the repo's `CLAUDE.md`).
3. This app: `npm install`, then `npm start`, and open http://localhost:4200.

`npm start` proxies `/api` to `http://localhost:8080` (`proxy.conf.json`), so the browser only ever talks to localhost:4200 and no CORS setup is needed.

A new account has to verify its email before the API accepts it. The emulator sends no email: the "Check your inbox" page links to the emulator's list of verification links; open the newest one, then press "I've verified my email".

## Commands

| Command | What it does |
|---|---|
| `npm start` | Dev server with live reload (port 4200) |
| `npm run test:ci` | Runs the tests once (Vitest) |
| `npm test` | Runs the tests in watch mode |
| `npm run build` | Production build into `dist/frontend/browser` |
| `npm run api:generate` | Regenerates `src/app/api/schema.d.ts` from `../docs/openapi.yaml` |
| `npm run api:check` | Fails if `schema.d.ts` is out of date |
| `npm run format` | Formats the source with Prettier |

## How it is organised

- `src/app/api/`: the API's types (generated), thin `HttpClient` services, and the validation limits the forms mirror.
- `src/app/core/`: sign-in state, the HTTP interceptor (adds the token, handles 401, 403 "email not verified" and 429), and the route guards.
- `src/app/shared/`: form validators and small shared pieces.
- `src/app/features/`: the screens: `auth/` and `warbands/`.
- `src/environments/`: development settings (emulator) and production settings (to be filled in before deploying).

The API types come from the OpenAPI spec, so when the API changes: update `docs/openapi.yaml`, run `npm run api:generate`, and fix whatever TypeScript now complains about.
