# Monthly Goal Tracker

Monthly Goal Tracker is being developed as an AI-assisted goal tracking system
that helps users make better daily decisions, not just record past activity.

## Project Goal

The goal of this project is to build a monthly goal tracker that can evolve from
a single-user prototype into a scalable multi-user system with authentication
and user-specific data.

The current version is a prototype. It supports the core monthly tracking
workflow, while multi-user authentication and AI-assisted insights are planned
for future iterations.

## Why This Project

This is not just a tracking tool. It is intended to become a decision-support
system for personal goal management.

By analyzing daily completion patterns and notes, the system is designed to help:

- Identify which goals are consistently failing.
- Understand behavioral patterns behind missed days.
- Adjust goals based on realistic capacity.
- Make better decisions for the next day and next month.

## Features

- Create and edit monthly goals.
- Carry active goals into the next month.
- Track daily completion across up to five active goals.
- Save daily notes with completion history.
- Review monthly progress with a dashboard and chart.

## Development Approach

This project uses AI-assisted development workflows to:

- Accelerate prototyping and iteration speed.
- Explore multiple design options quickly.
- Improve code quality through iterative refinement.

AI is used as a productivity tool, while system design and architecture
decisions remain developer-led.

The focus is on using AI to accelerate development without compromising
engineering judgment.

## AI-Assisted Insights

The system is designed to evolve with AI-assisted features, such as:

- Summarizing daily notes into actionable insights.
- Detecting patterns in goal completion.
- Suggesting goal adjustments based on user behavior.
- Providing simple daily recommendations.

The focus is not on complex AI, but on practical decision support.

## Tech Stack

- Backend: Go, Gin
- Database: PostgreSQL
- Frontend: React, Vite
- Infrastructure: Docker, EC2, Caddy

## Run Locally

### Prerequisites

- Docker with Compose
- Go 1.25
- Node.js 22
- pnpm 10.33.2

### Setup

1. Create local environment variables.

   If you use nvm, select the same Node.js major version as CI first.

   ```sh
   nvm use
   corepack enable
   cp .env.example .env
   ```

   `.env` is ignored by Git. Keep machine-specific values there.

2. Start PostgreSQL.

   ```sh
   sh scripts/dev-db.sh
   ```

3. Start the API in a second terminal.

   ```sh
   sh scripts/dev-api.sh
   ```

   The API listens on `http://127.0.0.1:8080` by default.

4. Start the web app in a third terminal.

   ```sh
   sh scripts/dev-web.sh
   ```

   The web app listens on `http://127.0.0.1:5173` and proxies `/api` requests to
   the local API.

## Verification

Run the backend tests, frontend tests, and frontend production build:

```sh
sh scripts/verify.sh
```

Quick API checks while the DB and API are running:

```sh
curl -s http://127.0.0.1:8080/api/health
curl -s http://127.0.0.1:8080/api/months/2026-05
```

The frontend build can emit a Vite chunk-size warning. That warning does not
fail the build.

## Deployment

This repository includes EC2 deployment examples under `deploy/`:

- Caddy serves the Vite production build over HTTPS.
- Caddy protects the whole site with Basic Auth.
- Caddy proxies `/api/*` to the Go API on `127.0.0.1:8080`.
- PostgreSQL runs on the EC2 host through Docker Compose with its port bound to
  loopback only.
- The Go API runs as a systemd service and keeps `APP_HOST=127.0.0.1`.

Start with `deploy/README.md` when preparing an EC2 deployment.

## Notes

- Application-level authentication is not implemented yet.
- AI-assisted product insights are planned and are not part of the current
  implementation.
- The API is bound to loopback by default and should not be exposed directly to
  the public internet.
- If the app is deployed before authentication is added, place it behind HTTPS
  and Basic Auth.
- Use strong, server-only PostgreSQL credentials for any deployed environment.
- Do not commit `.env` or other machine-specific configuration files.

## Useful Commands

```sh
docker compose ps
docker compose logs postgres
docker compose down
```
