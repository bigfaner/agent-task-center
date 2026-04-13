.PHONY: server web build dev-server dev-web dev lint vet fmt format check

server:
	cd server && go build -o bin/server ./cmd/server

web:
	cd web && npm run build

build: web server

dev-server:
	cd server && go run ./cmd/server

dev-web:
	cd web && npm run dev

dev:
	$(MAKE) -j2 dev-server dev-web

vet:
	cd server && golangci-lint run ./...

fmt:
	cd server && golangci-lint fmt ./...

lint:
	cd web && npm run lint

format:
	cd web && npm run format

check: vet lint
	@echo "All checks passed"
