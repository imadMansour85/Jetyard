.PHONY: help dev backend frontend nats install-backend install-frontend clean

help:
	@echo "Jetyard - Available commands:"
	@echo "  make dev              - Start all services with Docker Compose"
	@echo "  make backend          - Run backend locally"
	@echo "  make frontend         - Run frontend locally"
	@echo "  make nats             - Start only NATS server"
	@echo "  make install-backend  - Install backend dependencies"
	@echo "  make install-frontend - Install frontend dependencies"
	@echo "  make clean            - Stop all services and clean up"

dev:
	docker-compose up --build

backend:
	cd backend && go run .

frontend:
	cd frontend && npm run dev

nats:
	docker run -p 4222:4222 -p 8222:8222 nats:latest -js -m 8222

install-backend:
	cd backend && go mod download

install-frontend:
	cd frontend && npm install

clean:
	docker-compose down -v
	rm -rf backend/jetyard
	rm -rf frontend/.next frontend/node_modules
