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
