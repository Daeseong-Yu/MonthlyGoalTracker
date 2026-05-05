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

## Pre-Deployment Setup

1. Prepare the EC2 instance.

   Install Git, Docker with the Compose plugin, Caddy, curl, rsync, and the
   OpenSSH server. Assign an Elastic IP and point your domain A record to that
   IP. Go, Node.js, and pnpm are not required on the EC2 host when GitHub
   Actions performs production builds.

2. Clone the deployment templates and create the application directories.

   ```sh
   export DEPLOY_USER=<deploy-user>
   sudo install -d -m 0755 -o "$DEPLOY_USER" -g "$DEPLOY_USER" /opt/monthly-goal-tracker
   git clone <repo-url> /opt/monthly-goal-tracker
   mkdir -p /opt/monthly-goal-tracker/backend/releases
   mkdir -p /opt/monthly-goal-tracker/frontend/releases
   ```

   The EC2 host keeps the deployment examples and Compose file from the
   repository, but it does not build the application from source.

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
   sudo systemctl enable monthly-goal-api
   ```

   The first service start requires a deployed API binary at
   `/opt/monthly-goal-tracker/backend/monthly-goal-api`. Let the first backend
   deployment install the binary and start the service, or copy a locally built
   binary to that path before running `sudo systemctl start monthly-goal-api`.

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

## GitHub Actions CI/CD

The repository uses separate workflows for backend CI, frontend CI, and
production deployment.

- `backend-ci.yml`: runs Go tests and a Linux binary build when `backend/**`,
  the backend workflow, the deploy workflow, or `scripts/verify.sh` changes on
  `feature/**` or `develop`, and on PRs to `develop` or `main`.
- `frontend-ci.yml`: runs frontend tests and production build when
  `frontend/**`, the frontend workflow, the deploy workflow, or
  `scripts/verify.sh` changes on `feature/**` or `develop`, and on PRs to
  `develop` or `main`.
- `deploy.yml`: runs on `main` pushes and deploys only the changed component.
  Manual dispatch can deploy `all`, `backend`, or `frontend`.
  Changes outside `backend/**` and `frontend/**` do not automatically replace
  production artifacts.

Recommended branch flow:

```text
feature/* -> develop -> main -> production deploy
```

Configure these GitHub secrets before enabling production deployment:

```text
EC2_HOST
EC2_USER
EC2_SSH_PRIVATE_KEY
EC2_SSH_KNOWN_HOSTS
EC2_SSH_PORT # optional, defaults to 22
```

The deploy user should own `/opt/monthly-goal-tracker` so CI can replace build
artifacts without broad passwordless sudo. It only needs passwordless sudo for
the API restart:

```text
<deploy-user> ALL=(root) NOPASSWD: /usr/bin/systemctl restart monthly-goal-api
```

Adjust `/usr/bin/systemctl` if `command -v systemctl` returns a different path
on your EC2 image.

Backend deployments build `linux/amd64` and `linux/arm64` binaries in GitHub
Actions. The EC2 host selects the right binary with `uname -m`, installs it to
`/opt/monthly-goal-tracker/backend/monthly-goal-api`, restarts systemd, checks
`http://127.0.0.1:8080/api/health`, and restores the previous binary if the
health check fails.

Each deployment uses a release name derived from the commit SHA, workflow run,
and run attempt so rerunning the same commit creates a fresh release directory.

Frontend deployments build `frontend/dist` in GitHub Actions, upload it to a
release directory under `/opt/monthly-goal-tracker/frontend/releases`, switch
`/opt/monthly-goal-tracker/frontend/dist` to the new release, and keep the most
recent five releases.

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
