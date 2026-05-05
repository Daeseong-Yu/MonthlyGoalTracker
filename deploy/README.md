# EC2 Deployment

This guide describes an EC2 deployment for the current version of Monthly Goal
Tracker. Application-level authentication is not implemented yet, so the Go API
stays on loopback and the site is protected with HTTPS Basic Auth.

## Architecture

```text
Browser
  -> HTTPS domain
  -> Caddy Basic Auth
  -> React static files
  -> /api/* reverse proxy to 127.0.0.1:8080
  -> PostgreSQL on 127.0.0.1:5433
```

Recommended EC2 exposure:

- Open `80/tcp` and `443/tcp` to the internet.
- Restrict `22/tcp` to your own IP.
- Do not expose `8080`, `5432`, or `5433`.

## Files

- `Caddyfile.example`: HTTPS, Basic Auth, static frontend, and API proxy.
- `docker-compose.postgres.yml`: production-style PostgreSQL service bound to
  loopback.
- `monthly-goal-api.service.example`: systemd service for the Go API.
- `env.production.example`: placeholder environment values for the EC2 host.

## Server Layout

Use a generic application directory such as:

```sh
/opt/monthly-goal-tracker
```

Expected build artifacts:

```text
/opt/monthly-goal-tracker/backend/monthly-goal-api
/opt/monthly-goal-tracker/frontend/dist
/etc/monthly-goal-tracker/api.env
/etc/caddy/Caddyfile
```

## Deployment Steps

1. Prepare the EC2 instance.

   Install Git, Docker with the Compose plugin, Caddy, Go, Node.js, and pnpm.
   Assign an Elastic IP and point your domain A record to that IP.

2. Clone and build the app.

   ```sh
   git clone <repo-url> /opt/monthly-goal-tracker
   cd /opt/monthly-goal-tracker
   cd backend
   go build -o monthly-goal-api ./cmd/api
   cd ../frontend
   pnpm install --frozen-lockfile
   pnpm build
   ```

3. Configure PostgreSQL.

   Create the server-only environment directory, copy the example, and replace
   every placeholder.

   ```sh
   sudo mkdir -p /etc/monthly-goal-tracker
   sudo cp deploy/env.production.example /etc/monthly-goal-tracker/postgres.env
   sudoedit /etc/monthly-goal-tracker/postgres.env
   cd /opt/monthly-goal-tracker
   docker compose --env-file /etc/monthly-goal-tracker/postgres.env \
     -f deploy/docker-compose.postgres.yml up -d
   ```

4. Configure the API service.

   Create `/etc/monthly-goal-tracker/api.env` with production values:

   ```sh
   APP_HOST=127.0.0.1
   APP_PORT=8080
   DATABASE_URL=postgres://monthly_goal_tracker:<strong-password>@127.0.0.1:5433/monthly_goal_tracker?sslmode=disable
   ```

   Install the systemd unit from `monthly-goal-api.service.example`, then run:

   ```sh
   sudo useradd --system --no-create-home --shell /usr/sbin/nologin monthlygoal
   sudo cp deploy/monthly-goal-api.service.example /etc/systemd/system/monthly-goal-api.service
   sudo systemctl daemon-reload
   sudo systemctl enable --now monthly-goal-api
   sudo systemctl status monthly-goal-api
   ```

5. Configure Caddy.

   Generate a Basic Auth password hash on the server:

   ```sh
   caddy hash-password
   ```

   Copy `Caddyfile.example` to `/etc/caddy/Caddyfile`, replace `example.com`,
   `www.example.com`, `app-user`, and `<caddy-hashed-password>`, then run:

   ```sh
   sudo cp deploy/Caddyfile.example /etc/caddy/Caddyfile
   sudoedit /etc/caddy/Caddyfile
   sudo caddy validate --config /etc/caddy/Caddyfile
   sudo systemctl reload caddy
   ```

## Smoke Checks

After DNS and HTTPS are active:

```sh
curl -I https://example.com
curl -u app-user:<password> https://example.com/api/health
```

Then verify the main workflow in the browser:

- Sign in with the Basic Auth account.
- Open the current month.
- Create a goal.
- Toggle a daily completion check.
- Save a daily memo.
- Use the month preparation flow.

## Operations Notes

- Keep real secrets only in `/etc/monthly-goal-tracker/*.env`.
- Keep PostgreSQL bound to loopback.
- Back up the Docker volume before replacing or rebuilding the EC2 instance.
- Treat Basic Auth as a temporary access guard, not the final application
  authentication model.
