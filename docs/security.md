# Security model

This page explains what Jetyard stores, which secrets it needs, how the dashboard authenticates people, and how to deploy it safely. It describes the code as it is today, including the known gaps.

## What the dashboard can do to your NATS

The backend opens **one NATS connection** with the credentials you configure (`NATS_URL` or the Settings page). Everything the dashboard does, it does with those credentials:

- list, create and delete streams, consumers, KV buckets and keys
- publish to any subject those credentials allow
- read stored messages and KV values
- read `/varz` and `/connz` from the NATS monitoring port

Give the dashboard a NATS user whose permissions match what you want people to manage through it. Its reach is the reach of those credentials.

Dashboard roles control who can do what through the UI:

| Role | Can |
|---|---|
| `admin` | everything, including deletes, publishing, creating accounts and users, managing dashboard users |
| `viewer` | read-only. A viewer can still **read message payloads and KV values**, so treat viewer access as access to your data |

## What is stored on disk

The backend writes three JSON files, by default under `/data` (the Docker volume). Each path can be changed with an environment variable. Files are created with mode `0600` and directories with `0700`, owned by the user the backend runs as.

| File | Variable | Contents | Protection |
|---|---|---|---|
| `/data/auth-users.json` | `AUTH_USERS_FILE` | Dashboard usernames, roles, password hashes | bcrypt (cost 10). Plain passwords are never stored |
| `/data/nats-store.json` | `NATS_STORE_FILE` | NATS accounts created in the dashboard (name, public key, account JWT, **account seed**) and NATS users (name, public key, account, user JWT, permissions) | Account seeds are encrypted with AES-256-GCM when `ENCRYPTION_KEY` is set, and plain text otherwise. **User seeds are never stored**: they are returned once, in the `.creds` download at creation |
| `/data/nats-settings.json` | `NATS_SETTINGS_FILE` | NATS connection saved from the Settings page: URL, auth type, username, **password, token, TLS client key**, CA and client certificate | Password, token and client key are encrypted with AES-256-GCM when `ENCRYPTION_KEY` is set, and plain text otherwise. Only written if you save settings in the UI |

Nothing else is written. The backend makes no outbound connections except to your NATS server and its monitoring port. The frontend loads the Inter font from Google Fonts; self-host it if the browser has no internet access.

**Backups of `/data` contain password hashes and, without `ENCRYPTION_KEY`, account seeds and NATS passwords in plain text.** Encrypt backups, and store `ENCRYPTION_KEY` separately from them.

## Secrets the backend reads from the environment

| Variable | What it protects | If it leaks |
|---|---|---|
| `JWT_SECRET` | Signs dashboard sessions (HMAC-SHA256). At least 32 characters | Anyone can forge a session for any user, including an admin. Rotate it: all sessions end |
| `OPERATOR_NKEY_SEED` | Signs the account JWTs the dashboard creates | Anyone can sign accounts your NATS servers trust. Create a new operator, re-sign accounts, update the servers |
| `ENCRYPTION_KEY` | Encrypts account seeds and NATS credentials at rest (32 bytes, base64) | Combined with a copy of `/data`, exposes those seeds and credentials. **If you lose it, encrypted seeds can't be recovered** |

Recommendations:

- Inject them from a secrets manager: Docker secrets, a Kubernetes `Secret`, Vault, or `fly secrets set`. Don't put them in `docker-compose.yml`, in an image, or in a file committed to git (`.env` is in `.gitignore`).
- Treat `OPERATOR_NKEY_SEED` as the most sensitive value. Limit who can read the backend's environment.
- If `JWT_SECRET` is unset, the backend generates a random one at every start. That is safe, but it logs everyone out on each restart.
- `OPERATOR_NKEY_SEED` is optional. Without it the Accounts and Users pages are disabled, and the dashboard holds no signing key.

## Dashboard authentication

- **First admin.** `POST /api/auth/register` only works while no dashboard user exists, then closes for good. Until you register, anyone who can reach the backend can become admin, so register right after the first start, before exposing it to a network. More users are created by an admin from the dashboard.
- **Passwords** must be 12 to 72 characters (bcrypt ignores bytes beyond 72). Changing your own password requires the current one.
- **Sessions** last 12 hours, with no refresh, and are signed with `JWT_SECRET`. Logging out deletes the token in the browser only: a copied token stays valid until it expires. To end every session, rotate `JWT_SECRET`.
- **Token storage.** The browser keeps the token in `localStorage` and sends it as `Authorization: Bearer`. The live-updates stream (`/api/events`) passes it as `?access_token=` because `EventSource` can't set headers. The backend redacts it from its logs, but a reverse proxy logs it unless you configure it not to (see below).
- **CORS.** Only origins listed in `ALLOWED_ORIGINS` can call the API from a browser.
- **No brute-force protection.** The backend does not rate-limit `POST /api/auth/login`. Put the dashboard behind a reverse proxy that does (examples below), or keep it on a private network or VPN.
- `ADMIN_TOKEN`, `LICENSE_KEY` and `DEV_SKIP_AUTH` no longer exist. They belonged to the license server that was removed for the open-source release. If they are still in your `.env`, delete them.

## NATS accounts and users

- Account JWTs are signed by your operator but **not pushed to your NATS servers**. Add them to your resolver yourself (`GET /api/accounts/{name}/jwt` returns the JWT).
- A user's seed is shown once, in the `.creds` file at creation. Deleting a user in the dashboard **does not revoke** its `.creds`: NATS keeps accepting it until the user is revoked in the account JWT.
- Accounts created before account seeds were persisted can't issue new users. The Accounts page flags them.

## Network exposure

| Port | Service | Expose to |
|---|---|---|
| 3000 | Dashboard: UI and API in the single Docker image | Users, through the HTTPS reverse proxy |
| 3000 + 8080 | Frontend and backend API, in the development two-container setup | Users' browsers, through the same reverse proxy |
| 4222 | NATS client port | The backend |
| 8222 | NATS monitoring | **The backend only.** The monitoring endpoint has no authentication |

## TLS

### Dashboard to NATS

On the Settings page, enable TLS. Paste the CA certificate if your server uses a private CA, and a client certificate and key for mutual TLS. The NATS server must have a `tls {}` block. The test button checks the connection before anything is saved.

**Known limitation:** when the NATS connection requires TLS, the backend calls the monitoring port over `https://`, without the CA from the Settings page. NATS serves monitoring over plain HTTP unless you set `https_port`, so Server Info and Subscriptions fail on TLS-only setups. This is tracked as a bug.

### Browser to dashboard: reverse proxy with HTTPS

With the single Docker image, proxy everything to port 3000: the UI and the API share one origin, so there is nothing else to set. `docker-compose.prod.yml --profile tls` does this with Caddy (see [install.md](install.md)).

With the development two-container setup, serve the frontend and the API from one HTTPS origin, then build the frontend with that origin as its API URL and allow it in CORS:

```bash
NEXT_PUBLIC_API_URL=https://nats.example.com   # frontend build arg, inlined at build time
ALLOWED_ORIGINS=https://nats.example.com        # backend
```

The examples below use the two-container layout. For the single image, replace both `backend:8080` and `frontend:3000` with `jetyard:3000`.

**Caddy** (automatic HTTPS, streams Server-Sent Events without buffering):

```caddyfile
nats.example.com {
	@api path /api/* /health /version
	reverse_proxy @api backend:8080
	reverse_proxy frontend:3000

	# The live-updates URL carries the session token in its query string.
	log {
		format filter {
			wrap console
			fields {
				request>uri query {
					replace access_token REDACTED
				}
			}
		}
	}
}
```

For login rate limiting with Caddy, add a rate-limit plugin such as `caddy-ratelimit`, or put the dashboard behind your SSO proxy.

**nginx**, with buffering off for the event stream and a login rate limit:

```nginx
limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;

# Log the path without the query string: /api/events carries the session token.
log_format noquery '$remote_addr [$time_local] "$request_method $uri" $status $body_bytes_sent';

server {
    listen 443 ssl;
    server_name nats.example.com;
    ssl_certificate     /etc/ssl/nats.example.com.crt;
    ssl_certificate_key /etc/ssl/nats.example.com.key;
    access_log /var/log/nginx/jetyard.log noquery;

    location = /api/auth/login {
        limit_req zone=login burst=5 nodelay;
        proxy_pass http://backend:8080;
    }

    location = /api/events {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_read_timeout 1h;
    }

    location ~ ^/(api/|health$|version$) {
        proxy_pass http://backend:8080;
    }

    location / {
        proxy_pass http://frontend:3000;
    }
}
```

The backend reuses an incoming `X-Request-Id` header, so you can pass your proxy's request ID (`proxy_set_header X-Request-Id $request_id;` in nginx) and match proxy and backend logs.

## Running the containers

- The published image (`Dockerfile` at the repo root) is distroless and runs as a non-root user (uid 65532), with no shell or package manager. The development images in `backend/` and `frontend/` still run as root: don't use them in production.
- Give the container a dedicated volume for `/data`, no host network and no extra capabilities.
- Use JSON logs in production (`LOG_FORMAT=json`). Every error response and log line carries a request ID that users can quote.
- Keep the dashboard on the same private network as NATS where you can, and expose only the reverse proxy.

## Reporting a vulnerability

See [SECURITY.md](../SECURITY.md).
