.PHONY: all frontend backend api cli clean dev-frontend

all: frontend backend

frontend:
	cd frontend && npm ci && npm run build

backend: api cli

api:
	go build -o systemet-ass ./cmd/api

cli:
	go build -o systemet-poll-cli ./cmd/cli

dev-frontend:
	cd frontend && npm run dev

clean:
	rm -f systemet-ass systemet-poll-cli
	rm -rf frontend/dist
