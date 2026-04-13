.PHONY: server web build dev-server dev-web dev

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
