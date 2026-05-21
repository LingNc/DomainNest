.PHONY: build frontend dev clean run

# Build frontend and sync to embed directory
frontend:
	cd web && npm ci && npm run build
	rm -rf internal/static/dist/
	mkdir -p internal/static/dist
	cp -r web/dist/* internal/static/dist/

# Full production build (frontend + Go binary)
build: frontend
	go build -o domainnest ./cmd/server/

# Run development backend server
run:
	go run ./cmd/server/

# Development mode hint
dev:
	@echo "Terminal 1: make run"
	@echo "Terminal 2: cd web && npm run dev"

# Clean generated assets
clean:
	rm -rf internal/static/dist/
	rm -rf web/dist/
	rm -rf web/node_modules/.vite/
	rm -f domainnest
