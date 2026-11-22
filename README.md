# Jetyard

An open-source, self-hosted dashboard for NATS: streams, consumers, KV, accounts, users and monitoring, with live updates.

## Features

- 🎯 **Stream Management**: Create, view, and delete JetStream streams
- 👥 **Consumer Management**: Manage stream consumers with various policies
- 📨 **Message Operations**: Publish messages and view stream contents
- 📊 **Real-time Dashboard**: Live metrics and statistics
- ⚡ **SSE Integration**: Real-time updates without polling
- 🎨 **Modern UI**: Beautiful dark theme with glassmorphism design

## Architecture

```
┌─────────────┐         ┌─────────────┐         ┌─────────────┐
│   Next.js   │  HTTP   │   Golang    │  NATS   │    NATS     │
│  Frontend   │◄───────►│   Backend   │◄───────►│   Server    │
│  (Port 3000)│   SSE   │  (Port 8080)│         │  (Port 4222)│
└─────────────┘         └─────────────┘         └─────────────┘
```

### Backend (Golang)
- REST API for NATS management operations
- Server-Sent Events for real-time updates
- NATS JetStream client integration
- CORS-enabled for frontend communication

### Frontend (Next.js)
- Server-side rendering with Next.js 14
- TypeScript for type safety
- TailwindCSS for styling
- Real-time SSE connection with auto-reconnect

## Quick Start

With a NATS server already running (JetStream `-js` and monitoring `-m 8222` enabled):

```bash
docker run -d -p 3000:3000 \
  -e NATS_URL=nats://your-nats-host:4222 \
  -e JWT_SECRET="$(openssl rand -hex 32)" \
  -v jetyard-data:/data \
  ghcr.io/imadmansour85/jetyard:latest
```

Open http://localhost:3000 and create the first admin account. NATS credentials, HTTPS with Caddy, backups and upgrades: [docs/install.md](docs/install.md).

### Development

`make dev` (or `docker-compose up --build`) builds from source and starts its own NATS. To run the parts separately:

```bash
make nats                          # nats-server with -js -m 8222 in Docker
cd backend && go run .             # API on :8080
cd frontend && npm install && npm run dev   # UI on :3000
```

## API Endpoints

### Streams
- `GET /api/streams` - List all streams
- `GET /api/streams/{name}` - Get stream details
- `POST /api/streams` - Create stream
- `DELETE /api/streams/{name}` - Delete stream

### Consumers
- `GET /api/consumers?stream={name}` - List consumers for a stream
- `POST /api/consumers` - Create consumer
- `DELETE /api/consumers/{name}?stream={stream}` - Delete consumer

### Messages
- `POST /api/publish` - Publish message
- `GET /api/messages/{stream}?limit={n}` - Get recent messages

### SSE
- `GET /api/events` - Server-Sent Events stream

### Health & Stats
- `GET /health` - Health check
- `GET /version` - Backend version, git commit and build time (set with `-ldflags`, see `backend/buildinfo`)
- `GET /api/stats` - Overall statistics

## Configuration

### Backend Environment Variables
Copy `.env.example` to `.env` for Docker Compose. Never commit `.env`.

- `NATS_URL` - NATS server URL (default: `nats://localhost:4222`)
- `SERVER_PORT` - Backend server port (default: `8080`)
- `ALLOWED_ORIGINS` - Comma-separated origins allowed by CORS (default: `http://localhost:3000`). Must include the frontend's URL in any deployment.
- `JWT_SECRET` - Signs dashboard sessions, at least 32 characters. If unset, a random secret is generated on every start, which logs everyone out on restart.
- `OPERATOR_NKEY_SEED` - Operator NKey seed (`SO...`). Required for account and NATS user management.
- `NATS_MONITOR_PORT` - NATS monitoring port used for server info and connections (default: `8222`)
- `AUTH_USERS_FILE` / `NATS_STORE_FILE` / `NATS_SETTINGS_FILE` - Where dashboard users, accounts/users, and the NATS connection settings saved from the Settings page are persisted (defaults: `/data/auth-users.json`, `/data/nats-store.json`, `/data/nats-settings.json`). Set them when running the backend outside Docker, where `/data` usually doesn't exist.
- `LOG_FORMAT` - `text` (default) or `json` for structured logs in production. `LOG_LEVEL` - `debug`, `info` (default), `warn` or `error`. Every log line and every error response carries a request ID (`X-Request-Id`, reused from the caller if present), so a user can quote it when reporting a problem.
- `ENCRYPTION_KEY` - Optional, 32 bytes in base64 (`openssl rand -base64 32`). When set, NATS account seeds and the NATS password, token and TLS client key saved from the Settings page are encrypted at rest with AES-256-GCM. Keep it safe: without it those values can't be read back. When unset, they are stored in plain text in `0600` files. See [docs/security.md](docs/security.md).

### Frontend Environment Variables
- `NEXT_PUBLIC_API_URL` - Backend API URL, set at build time. Empty by default: the UI calls the API on its own origin, as in the Docker image. `npm run dev` uses `http://localhost:8080` from `frontend/.env.development`.

## Development

### Available Make Commands

```bash
make help              # Show all available commands
make dev               # Start all services with Docker
make backend           # Run backend locally
make frontend          # Run frontend locally
make nats              # Start only NATS server
make install-backend   # Install backend dependencies
make install-frontend  # Install frontend dependencies
make clean             # Stop services and clean up
```

## Project Structure

```
./
├── backend/
│   ├── config/         # Configuration management
│   ├── handlers/       # HTTP request handlers
│   ├── middleware/     # HTTP middleware
│   ├── models/         # Data models
│   ├── nats/           # NATS client wrapper
│   ├── main.go         # Application entry point
│   ├── go.mod          # Go dependencies
│   └── Dockerfile      # Backend container image
├── frontend/
│   ├── app/            # Next.js app directory
│   │   ├── page.tsx    # Dashboard
│   │   ├── streams/    # Streams management
│   │   ├── consumers/  # Consumers management
│   │   └── messages/   # Message operations
│   ├── components/     # React components
│   ├── lib/            # API client and utilities
│   ├── package.json    # Node dependencies
│   └── Dockerfile      # Frontend container image
├── docker-compose.yml  # Multi-container setup
├── Makefile            # Development commands
└── README.md           # This file
```

## Screenshots

### Dashboard
Real-time metrics and quick actions for NATS management.

### Streams
View and manage JetStream streams with detailed statistics.

### Consumers
Configure consumers with various delivery and acknowledgment policies.

### Messages
Publish messages and view recent stream contents.

## Technologies Used

### Backend
- **Go 1.21** - Programming language
- **NATS Go Client** - NATS JetStream integration
- **net/http** - HTTP server (Go 1.22+ router)
- **rs/cors** - CORS middleware

### Frontend
- **Next.js 14** - React framework
- **TypeScript** - Type safety
- **TailwindCSS** - Styling
- **Lucide React** - Icons
- **EventSource API** - SSE client

## Security

What the dashboard stores, which secrets it needs, and how to run it behind HTTPS: [docs/security.md](docs/security.md). To report a vulnerability, see [SECURITY.md](SECURITY.md).

## License

Jetyard is free and open-source software under the [GNU Affero General Public License v3.0 or later](LICENSE) (`AGPL-3.0-or-later`). There is no license key and no feature gate.

What that means in practice:

- **Running it on-prem for your own team**: nothing to do. Use it, modify it, deploy it on as many servers as you like.
- **Changing it and offering it to other people over a network** (for example as a hosted service): the AGPL requires you to make the source of your modified version available to those users.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for the dev setup, tests and the DCO sign-off. Questions go to [Discussions](../../discussions).
