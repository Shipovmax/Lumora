# Lumora

> Understand what matters.

Lumora is an AI-powered personal briefing platform that helps people stay informed without information overload.

Instead of endless feeds, Lumora delivers concise, personalized briefings twice a day, explaining not only **what happened**, but also **why it matters** and **what it means for each individual user**.

---

## Vision

The internet doesn't need another news feed.

People don't need more information.

People need a filter.

Lumora adapts information to the user—not the other way around.

---

## Core Principles

- Save time instead of consuming it
- No infinite scrolling
- No social network
- No AI chat
- No engagement traps
- Only meaningful information

---

## How it works

Every day Lumora:

1. Collects information from trusted sources.
2. Removes duplicates.
3. Groups related events.
4. Estimates importance.
5. Personalizes content using the user's context.
6. Generates a concise morning and evening briefing.

Every event contains four blocks:

- What happened
- Why it happened
- What changes
- What it means for you

---

## Example Briefing

☀️ Morning Briefing

AI — 2 events

Economy — 3 events

Crypto — 2 events

World — 1 event

Reading time: ~6 minutes

---

## MVP

The first version focuses on validating one hypothesis:

> Would people replace dozens of information sources with two personalized briefings every day?

The MVP includes:

- Authentication
- User context
- Source management
- Daily briefing generation
- AI summarization
- Push notifications for critical events

---

## Long-term Vision

Lumora aims to become the personal information layer between people and the internet.

Instead of opening ten different apps every morning, users open one.

Read for 5 minutes.

Understand everything important.

Close the app.

---

## Philosophy

We don't compete for attention.

We give attention back.

---

## Architecture

Backend is a modular monolith written in Go. See [ARCHITECTURE.md](ARCHITECTURE.md) for the full design: layering, module boundaries, tech stack, and the mapping between the development stages below and the codebase.

## Getting Started

Requirements: Docker + Docker Compose, Go 1.26+ (for local, non-container development).

```bash
cp .env.example .env
docker compose -f deployments/docker-compose.yml up --build -d
```

This starts PostgreSQL, Redis, the REST API (`cmd/api`) and the background worker (`cmd/worker`).

Apply database migrations:

```bash
make migrate-up
```

Check that everything is up:

```bash
curl http://localhost:8080/healthz   # liveness
curl http://localhost:8080/readyz    # readiness: checks Postgres + Redis
```

Shortcuts are available via `Makefile`: `make up`, `make down`, `make logs`, `make run-api`, `make run-worker`, `make migrate-up`, `make migrate-down`, `make sqlc`, `make test`, `make test-integration`.

## API

Full OpenAPI 3.0 spec, with a request/response example for every endpoint and status code: [`docs/openapi.yaml`](docs/openapi.yaml). Paste it into [editor.swagger.io](https://editor.swagger.io) or any OpenAPI viewer for an interactive read.

All endpoints are under `/api/v1`. Auth (Этап 2):

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/auth/register` | — | Create a user, returns access + refresh tokens |
| POST | `/api/v1/auth/login` | — | Exchange email/password for tokens |
| POST | `/api/v1/auth/refresh` | — | Rotate a refresh token for a new token pair |
| POST | `/api/v1/auth/logout` | — | Revoke a refresh token (idempotent) |
| GET | `/api/v1/auth/me` | Bearer access token | Current user's profile |

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"supersecret1"}'
```

Access tokens are short-lived JWTs (`ACCESS_TOKEN_TTL`, default 15m). Refresh tokens are opaque, stored hashed, rotated on every `/refresh` call, and revocable via `/logout`.

User profile (Этап 3):

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/user/profile` | Bearer access token | Current user's profile (created empty on first access) |
| PUT | `/api/v1/user/profile` | Bearer access token | Replace the profile (name, country, language, profession, interests, topics) |

```bash
curl -X PUT http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Ada Lovelace","country":"UK","language":"en","profession":"Mathematician","interests":["computing"],"topics":["ai"]}'
```

User context (Этап 4) — free-form text used by the AI when generating briefings, editable at any time:

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/api/v1/context` | Bearer access token | Current user's AI context (created empty on first access) |
| PUT | `/api/v1/context` | Bearer access token | Replace the context content (max 4000 characters) |

```bash
curl -X PUT http://localhost:8080/api/v1/context \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"content":"Interested in deep tech and AI research, skip celebrity news."}'
```

Sources (Этап 5) — RSS/YouTube/Telegram sources a user subscribes to:

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/sources` | Bearer access token | Add a source (`type`: `rss`\|`youtube`\|`telegram`, `name`, `url`) |
| GET | `/api/v1/sources` | Bearer access token | List the current user's sources |
| PATCH | `/api/v1/sources/{id}` | Bearer access token | Enable/disable a source (`{"enabled": true\|false}`) |
| DELETE | `/api/v1/sources/{id}` | Bearer access token | Remove a source |

```bash
curl -X POST http://localhost:8080/api/v1/sources \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"type":"rss","name":"Hacker News","url":"https://news.ycombinator.com/rss"}'
```

For `type: "youtube"`, `url` must be the channel's public Atom feed, e.g. `https://www.youtube.com/feeds/videos.xml?channel_id=<CHANNEL_ID>`. For `type: "telegram"`, `url` must be a public channel link, e.g. `https://t.me/<channel>`.

### Importing publications (Этап 6)

`internal/ingest` fetches new publications for a source, deduplicates them (`posts.source_id + external_id` is unique) and stores them. There is no HTTP endpoint and no automatic scheduler yet — import runs as an asynq task (`ingest:fetch`, `cmd/worker`) that must be enqueued manually for now, e.g. via [`asynqmon`](https://github.com/hibiken/asynqmon) or a small script, with a JSON payload:

```json
{"source_id": "<source uuid>"}
```

Periodic auto-polling of enabled sources is intentionally deferred (see `ARCHITECTURE.md` §7).

### Event processing (Этап 7)

After a successful import, the worker automatically enqueues `pipeline:process` for the newly saved publications — `internal/pipeline` clusters them into events (near-duplicate/related publications from different sources become one event), assigns a topic (`ai`/`economy`/`crypto`/`world`/`other`) and an importance score based on how many independent sources cover the event. No ML/embeddings and no HTTP endpoint yet — see `ARCHITECTURE.md` §7 for the clustering approach and its known simplifications.

### AI explanations (Этап 8)

`internal/ai` generates a personalized four-block explanation (what happened / why / what changed / what it means for you) for an `(event, user)` pair, using Anthropic Claude (`claude-opus-5`) and that user's context (Этап 4). Requires `ANTHROPIC_API_KEY` in the environment (`cmd/worker` only). No HTTP endpoint. As of Этап 9, `internal/briefing` calls this directly (in-process, not via a separate queued task) for events selected into a user's briefing — see below.

### Briefing generation (Этап 9)

`internal/briefing` builds a morning/evening briefing for each user: events relevant to them (covered by their own sources), not already included in a previous briefing, up to the 10 most important, each with an AI explanation (generated on demand if missing). No HTTP endpoint yet.

A built-in scheduler (`asynq.Scheduler`, UTC, no per-user timezone yet) fires automatically at **08:00 and 20:00 UTC**, dispatching a `briefing:build` task per user who has at least one source — no manual trigger needed. See `ARCHITECTURE.md` §7 for the relevance rule, dedup logic, and the known limitation of running the scheduler inside `cmd/worker` (fine for a single worker instance; would need to move to its own process if `cmd/worker` is scaled to multiple replicas).

### Push notifications (Этап 10)

`internal/notification` registers device tokens and pushes a notification only for events genuinely worth interrupting the user for — no notification spam:

| Method | Path | Auth | Description |
|---|---|---|---|
| POST | `/api/v1/notifications/devices` | Bearer access token | Register/refresh a device token (`platform`: `ios`\|`android`\|`web`, `token`) |
| DELETE | `/api/v1/notifications/devices` | Bearer access token | Remove a device token (only if it belongs to the caller) |

```bash
curl -X POST http://localhost:8080/api/v1/notifications/devices \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"platform":"android","token":"<fcm-registration-token>"}'
```

After building a briefing, the worker automatically enqueues `notification:push` for events with importance ≥ 60 (roughly: confirmed by 3+ independent sources — see `ARCHITECTURE.md` §7 for how importance is computed). Delivery goes through Firebase Cloud Messaging (HTTP v1 API); a token FCM reports as unregistered/invalid is removed automatically so it isn't retried. Requires `FCM_PROJECT_ID` and `FCM_CREDENTIALS_FILE` in the environment (`cmd/worker` only — see `.env.example`); without them the worker still starts, only `notification:push` fails until configured.
