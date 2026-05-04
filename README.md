# Monthly Goal Tracker

Monthly Goal Tracker is a local-first monthly habit and goal tracker with a Go
API, PostgreSQL storage, and a Vite React frontend.

## Prerequisites

- Docker with Compose
- Go 1.25
- pnpm 10

## Local Setup

1. Create local environment variables.

   ```sh
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

   The API listens on `http://127.0.0.1:8080` by default. It requires a
   loopback host while authentication is not implemented.

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

## Useful Commands

```sh
docker compose ps
docker compose logs postgres
docker compose down
```
