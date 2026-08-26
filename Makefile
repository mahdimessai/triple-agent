.PHONY: dev server web build test verify format format-check docker-dev docker

GOFUMPT := go tool gofumpt

dev:
	@make -j 2 server web

server:
	go run ./server/cmd/tripleagent

web:
	bun run --cwd web dev

build:
	bun run --cwd web build

test:
	go test ./server/...
	bun --cwd web test

verify:
	go test ./server/...
	bun run --cwd web verify

format:
	$(GOFUMPT) -w .

format-check:
	@files=$$( $(GOFUMPT) -l . ); \
	if [ -n "$$files" ]; then \
		echo "Go files need formatting:"; \
		echo "$$files"; \
		exit 1; \
	fi

docker-dev:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

docker:
	docker compose up --build
