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

Check that everything is up:

```bash
curl http://localhost:8080/healthz   # liveness
curl http://localhost:8080/readyz    # readiness: checks Postgres + Redis
```

Shortcuts are available via `Makefile`: `make up`, `make down`, `make logs`, `make run-api`, `make run-worker`, `make migrate-up`, `make migrate-down`, `make test`.
