# Install

Jetyard ships as one Docker image that serves the dashboard and its API on port 3000. It connects to a NATS server you already run.

**You need:** Docker, and a NATS server (2.10 or later) with JetStream enabled (`-js`) and the monitoring port enabled (`-m 8222`), reachable from where the container runs.

## 1. Run it

```bash
docker run -d --name jetyard \
  -p 3000:3000 \
  -e NATS_URL=nats://your-nats-host:4222 \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  -v jetyard-data:/data \
  ghcr.io/imadmansour85/jetyard:latest
```

Open <http://localhost:3000>, then **create the first admin account straight away**: until you do, anyone who can reach the port can claim it.

If NATS requires authentication, add one of:

```bash
  -e NATS_USER=dashboard -e NATS_PASSWORD=...   # user and password
  -e NATS_TOKEN=...                             # token
```

TLS, client certificates and other auth methods are configured on the **Settings** page after login. Settings saved there take priority over the environment variables.

`JWT_SECRET` signs login sessions. Pass the same value on every start, or everyone is logged out when the container restarts. Store it in a secrets manager rather than in your shell history (see [security.md](security.md)).

## 2. Optional features

| Variable | What it enables |
|---|---|
| `OPERATOR_NKEY_SEED` | The Accounts and Users pages: the dashboard signs NATS account and user JWTs with this operator seed (`SO...`, from `nsc generate nkey --operator`) |
| `ENCRYPTION_KEY` | Encrypts account seeds and saved NATS credentials on disk. 32 bytes, base64: `openssl rand -base64 32`. Keep it: those values can't be read without it |
| `NATS_MONITOR_PORT` | Monitoring port if it isn't 8222 |
| `LOG_FORMAT` / `LOG_LEVEL` | `json` (the image default) or `text`; `debug`, `info`, `warn`, `error` |

[`.env.example`](../.env.example) lists every variable with a comment.

## 3. With Docker Compose

[`docker-compose.prod.yml`](../docker-compose.prod.yml) runs the same image with your settings in a `.env` file, and can add Caddy for HTTPS:

```bash
cp .env.example .env    # set JWT_SECRET, NATS_URL and NATS credentials
docker compose -f docker-compose.prod.yml up -d
```

For HTTPS with an automatic Let's Encrypt certificate: point a domain at the host, open ports 80 and 443, set `DOMAIN=nats.example.com` and `LISTEN=127.0.0.1:3000` in `.env`, then:

```bash
docker compose -f docker-compose.prod.yml --profile tls up -d
```

The plain `docker-compose.yml` is the development stack. It builds from source and starts its own NATS.

## Data and backups

Everything the dashboard stores lives in the `/data` volume: dashboard users, NATS accounts and users, and settings saved in the UI. [security.md](security.md) lists each file.

Back up the volume while the container is stopped, or at least between changes:

```bash
docker run --rm -v jetyard-data:/data -v "$PWD":/backup alpine \
  tar czf /backup/jetyard-data.tgz -C /data .
```

The backup contains password hashes and, without `ENCRYPTION_KEY`, NATS credentials in plain text. Encrypt it, and keep `ENCRYPTION_KEY` somewhere else.

## Upgrading

```bash
docker pull ghcr.io/imadmansour85/jetyard:latest
docker stop jetyard && docker rm jetyard
# run the same `docker run` command as before
```

Pin a version (`:0.1.0`) instead of `:latest` if you want to choose when to upgrade. Releases and their notes are on the GitHub Releases page.

**Upgrading from the old two-container setup** (`backend` + `frontend` images, before v0.1.0): the new image runs as a non-root user (uid 65532), and files written by the old backend image belong to root. Fix the ownership once before starting the new image on the same volume:

```bash
docker run --rm -v backend-data:/data alpine chown -R 65532:65532 /data
```

Also remove `LICENSE_KEY`, `LICENSE_SERVER_URL`, `DEV_SKIP_AUTH` and `ADMIN_TOKEN` from your environment. They no longer exist.

## Health checks

- `GET /health` returns 200 while the process is up. The image's Docker `HEALTHCHECK` uses it.
- `GET /version` returns the version, git commit and build time.

## From source

You need Go 1.23+, Node.js 20+ and a NATS server.

```bash
nats-server -js -m 8222 &

# backend on :8080
cd backend && go run .

# frontend dev server on :3000, calling the backend on :8080
cd frontend && npm install && npm run dev
```

To build the single binary that serves the UI itself:

```bash
(cd frontend && npm ci && NEXT_OUTPUT=export npm run build)
(cd backend && go build -o jetyard .)
STATIC_DIR=frontend/out SERVER_PORT=3000 ./backend/jetyard
```
